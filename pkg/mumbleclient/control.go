package mumbleclient

import (
	"context"
	"errors"
	"io"
	"net"
	"time"

	"github.com/dchote/go-mumble-server/pkg/mumble/protocol"
	"github.com/dchote/go-mumble-server/pkg/mumble/protocol/messages"
)

// maintain runs the control loop and optional reconnect backoff.
func (c *Client) maintain(ctx context.Context) {
	defer close(c.maintainDone)

	backoff := ReconnectMinDelay
	for {
		err := c.controlLoop(ctx)
		c.stopUDP()
		c.teardownConn()
		wasConnected := c.IsConnected()
		c.setConnected(false)

		if c.isClosed() {
			if wasConnected {
				c.emit(Disconnected{})
			}
			return
		}
		if wasConnected {
			if errors.Is(err, context.Canceled) {
				err = nil
			}
			c.emit(Disconnected{Err: err})
		}

		if ctx.Err() != nil || c.cfg.DisableReconnect {
			return
		}

		if !c.waitBackoff(ctx, backoff) {
			return
		}

		syncInfo, connErr := c.connectOnce(ctx)
		if connErr != nil {
			if c.isClosed() || ctx.Err() != nil {
				return
			}
			backoff = nextBackoff(backoff)
			continue
		}
		backoff = ReconnectMinDelay
		c.emitConnected(syncInfo)
		c.startUDP()
	}
}

func (c *Client) isClosed() bool {
	select {
	case <-c.closed:
		return true
	default:
		return false
	}
}

func (c *Client) waitBackoff(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-c.closed:
		return false
	case <-t.C:
		return true
	}
}

func (c *Client) teardownConn() {
	c.mu.Lock()
	conn := c.conn
	c.conn = nil
	c.mu.Unlock()
	if conn != nil {
		_ = conn.Close()
	}
}

// controlLoop reads TCP messages until error, close, or context cancel.
func (c *Client) controlLoop(ctx context.Context) error {
	handlers := c.buildHandlerTable()
	pingTicker := time.NewTicker(TCPPingInterval)
	defer pingTicker.Stop()

	type result struct {
		msgType protocol.MessageType
		payload []byte
		err     error
	}
	reads := make(chan result, 1)

	readCtx, cancelReads := context.WithCancel(ctx)
	defer cancelReads()

	go func() {
		for {
			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()
			if conn == nil {
				select {
				case reads <- result{err: ErrNotConnected}:
				case <-readCtx.Done():
				}
				return
			}
			_ = conn.SetReadDeadline(time.Now().Add(TCPReadTimeout))
			msgType, payload, err := protocol.ReadPacket(conn)
			select {
			case reads <- result{msgType: msgType, payload: payload, err: err}:
			case <-readCtx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()

	// Prime keepalive clock so the first interval fires from now.
	c.pingMu.Lock()
	c.lastPingSent = time.Now()
	c.pingMu.Unlock()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.closed:
			return nil
		case <-pingTicker.C:
			if err := c.sendPing(); err != nil {
				return err
			}
		case r := <-reads:
			if r.err != nil {
				if c.isClosed() || ctx.Err() != nil {
					return nil
				}
				if ne, ok := r.err.(net.Error); ok && ne.Timeout() {
					return errors.New("mumbleclient: read timeout")
				}
				if errors.Is(r.err, io.EOF) {
					return io.EOF
				}
				return r.err
			}
			// Any packet refreshes the read deadline on the next iteration.
			if err := handlers.Dispatch(r.msgType, r.payload, c); err != nil {
				return err
			}
		}
	}
}

func (c *Client) buildHandlerTable() protocol.HandlerTable {
	t := protocol.NewHandlerTable()
	t[protocol.MessagePing] = c.onPing
	t[protocol.MessageCryptSetup] = c.onCryptSetup
	t[protocol.MessageChannelState] = c.onChannelState
	t[protocol.MessageChannelRemove] = c.onChannelRemove
	t[protocol.MessageUserState] = c.onUserState
	t[protocol.MessageUserRemove] = c.onUserRemove
	t[protocol.MessageTextMessage] = c.onTextMessage
	t[protocol.MessagePermissionDenied] = c.onPermissionDenied
	t[protocol.MessageUDPTunnel] = c.onUDPTunnel
	t[protocol.MessageReject] = c.onReject
	return t
}

func (c *Client) sendPing() error {
	ts := uint64(time.Now().UnixMilli())
	msg := &messages.Ping{Timestamp: ts, TCPPackets: 1}
	c.mu.RLock()
	crypt := c.crypt
	c.mu.RUnlock()
	if crypt != nil {
		msg.Good = crypt.Good
		msg.Late = crypt.Late
		msg.Lost = crypt.Lost
		msg.Resync = crypt.Resync
	}
	if err := c.WriteMessage(protocol.MessagePing, msg); err != nil {
		return err
	}
	c.pingMu.Lock()
	c.lastPingTimestamp = ts
	c.lastPingSent = time.Now()
	c.pingMu.Unlock()
	return nil
}

func (c *Client) onPing(_ protocol.MessageType, payload []byte, _ interface{}) error {
	msg, err := protocol.ReadMessage(protocol.MessagePing, payload)
	if err != nil || msg == nil {
		return nil
	}
	p, ok := msg.(*messages.Ping)
	if !ok {
		return nil
	}
	c.pingMu.Lock()
	sentTS := c.lastPingTimestamp
	sentAt := c.lastPingSent
	c.pingMu.Unlock()
	if p.Timestamp == sentTS && !sentAt.IsZero() {
		c.emit(PingRTT{RTT: time.Since(sentAt)})
	}
	return nil
}

func (c *Client) onCryptSetup(_ protocol.MessageType, payload []byte, _ interface{}) error {
	return c.applyCryptSetup(payload)
}

func (c *Client) onChannelState(_ protocol.MessageType, payload []byte, _ interface{}) error {
	msg, err := protocol.ReadMessage(protocol.MessageChannelState, payload)
	if err != nil || msg == nil {
		return nil
	}
	cs, ok := msg.(*messages.ChannelState)
	if !ok {
		return nil
	}
	ch, added := c.roster.ApplyChannelState(cs)
	if added {
		c.emit(ChannelAdd{Channel: ch})
	} else {
		c.emit(ChannelUpdate{Channel: ch})
	}
	return nil
}

func (c *Client) onChannelRemove(_ protocol.MessageType, payload []byte, _ interface{}) error {
	msg, err := protocol.ReadMessage(protocol.MessageChannelRemove, payload)
	if err != nil || msg == nil {
		return nil
	}
	cr, ok := msg.(*messages.ChannelRemove)
	if !ok {
		return nil
	}
	ch, _ := c.roster.RemoveChannel(cr.ChannelID)
	c.emit(ChannelRemove{ChannelID: cr.ChannelID, Channel: ch})
	return nil
}

func (c *Client) onUserState(_ protocol.MessageType, payload []byte, _ interface{}) error {
	msg, err := protocol.ReadMessage(protocol.MessageUserState, payload)
	if err != nil || msg == nil {
		return nil
	}
	us, ok := msg.(*messages.UserState)
	if !ok {
		return nil
	}
	u, added := c.roster.ApplyUserState(us)
	if added {
		c.emit(UserAdd{User: u})
	} else {
		c.emit(UserUpdate{User: u})
	}
	return nil
}

func (c *Client) onUserRemove(_ protocol.MessageType, payload []byte, _ interface{}) error {
	msg, err := protocol.ReadMessage(protocol.MessageUserRemove, payload)
	if err != nil || msg == nil {
		return nil
	}
	ur, ok := msg.(*messages.UserRemove)
	if !ok {
		return nil
	}
	u, _ := c.roster.RemoveUser(ur.Session)
	c.emit(UserRemove{
		Session: ur.Session,
		Actor:   ur.Actor,
		Reason:  ur.Reason,
		Ban:     ur.Ban,
		User:    u,
	})
	return nil
}

func (c *Client) onTextMessage(_ protocol.MessageType, payload []byte, _ interface{}) error {
	msg, err := protocol.ReadMessage(protocol.MessageTextMessage, payload)
	if err != nil || msg == nil {
		return nil
	}
	tm, ok := msg.(*messages.TextMessage)
	if !ok {
		return nil
	}
	c.emit(TextMessage{
		Actor:      tm.Actor,
		Sessions:   append([]uint32(nil), tm.Session...),
		ChannelIDs: append([]uint32(nil), tm.ChannelID...),
		TreeIDs:    append([]uint32(nil), tm.TreeID...),
		Message:    tm.Message,
	})
	return nil
}

func (c *Client) onPermissionDenied(_ protocol.MessageType, payload []byte, _ interface{}) error {
	msg, err := protocol.ReadMessage(protocol.MessagePermissionDenied, payload)
	if err != nil || msg == nil {
		return nil
	}
	pd, ok := msg.(*messages.PermissionDenied)
	if !ok {
		return nil
	}
	c.emit(PermissionDenied{
		Permission: pd.Permission,
		ChannelID:  pd.ChannelID,
		Session:    pd.Session,
		Reason:     pd.Reason,
		Type:       pd.Type,
		Name:       pd.Name,
	})
	return nil
}

func (c *Client) onUDPTunnel(_ protocol.MessageType, payload []byte, _ interface{}) error {
	c.emit(VoicePacket{
		Data:   append([]byte(nil), payload...),
		ViaUDP: false,
	})
	return nil
}

func (c *Client) onReject(_ protocol.MessageType, payload []byte, _ interface{}) error {
	msg, err := protocol.ReadMessage(protocol.MessageReject, payload)
	if err != nil || msg == nil {
		return errors.New("mumbleclient: rejected")
	}
	if m, ok := msg.(*messages.Reject); ok {
		return errors.New("mumbleclient: rejected: " + m.Reason)
	}
	return errors.New("mumbleclient: rejected")
}
