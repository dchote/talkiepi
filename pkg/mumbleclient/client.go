package mumbleclient

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/dchote/go-mumble-server/pkg/mumble/crypto"
	"github.com/dchote/go-mumble-server/pkg/mumble/protocol"
	"github.com/dchote/go-mumble-server/pkg/mumble/protocol/messages"
)

// Timing constants matching esp32-mumble client behavior.
const (
	TCPPingInterval   = 15 * time.Second
	TCPReadTimeout    = 45 * time.Second
	ReconnectMinDelay = 1 * time.Second
	ReconnectMaxDelay = 30 * time.Second

	UDPPingInitialInterval = 500 * time.Millisecond
	UDPPingInterval        = 5 * time.Second
	UDPPingInitialCount    = 5
)

// Crypto mode bits for Version.CryptoModes (bit0=lite, bit1=legacy, bit2=secure).
const (
	CryptoModeLite   uint32 = 1 << 0
	CryptoModeLegacy uint32 = 1 << 1
	CryptoModeSecure uint32 = 1 << 2
	// DefaultCryptoModes advertises lite and legacy (ESP32-compatible).
	DefaultCryptoModes = CryptoModeLite | CryptoModeLegacy
)

// Config configures a Mumble client connection.
type Config struct {
	Address     string // host:port
	Username    string
	Password    string
	Channel     string // join by name after sync; empty keeps server default
	InsecureTLS bool
	CertFile    string
	KeyFile     string
	// CryptoModes is the Version.CryptoModes bitmask. Zero means DefaultCryptoModes.
	CryptoModes uint32
	// Release is the Version.Release string. Empty defaults to "talkiepi".
	Release string
	// DisableReconnect prevents automatic reconnect after Dial.
	DisableReconnect bool
}

// Client is a Mumble control-plane client.
type Client struct {
	cfg Config

	mu           sync.RWMutex
	conn         *tls.Conn
	crypt        *crypto.CryptState
	roster       *Roster
	session      uint32
	channelID    uint32
	connected    bool
	transmitting bool
	voiceSeq     int64
	peerAddr     *net.UDPAddr

	writeMu sync.Mutex

	handlersMu sync.RWMutex
	handlers   []EventHandler
	events     chan Event

	udp *udpTransport

	closed          chan struct{}
	closeOnce       sync.Once
	cancel          context.CancelFunc
	maintainDone    chan struct{}
	maintainStarted bool

	pingMu            sync.Mutex
	lastPingTimestamp uint64
	lastPingSent      time.Time
}

// ErrNotConnected is returned when an operation requires an active session.
var ErrNotConnected = errors.New("mumbleclient: not connected")

// New creates a disconnected client.
func New(cfg Config) *Client {
	if cfg.CryptoModes == 0 {
		cfg.CryptoModes = DefaultCryptoModes
	}
	if cfg.Release == "" {
		cfg.Release = "talkiepi"
	}
	return &Client{
		cfg:          cfg,
		roster:       NewRoster(),
		crypt:        crypto.NewCryptState(crypto.ModeLegacy),
		events:       make(chan Event, 64),
		closed:       make(chan struct{}),
		maintainDone: make(chan struct{}),
	}
}

// Dial creates a client and connects. See Client.Dial.
func Dial(ctx context.Context, cfg Config) (*Client, error) {
	c := New(cfg)
	if err := c.Dial(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

// Dial establishes a TLS session, completes the Mumble handshake, joins the
// configured channel, and starts the control/UDP loops. On unexpected disconnect
// it reconnects with exponential backoff unless DisableReconnect is set or Close
// is called.
func (c *Client) Dial(ctx context.Context) error {
	select {
	case <-c.closed:
		return errors.New("mumbleclient: client closed")
	default:
	}

	ctx, cancel := context.WithCancel(ctx)
	c.mu.Lock()
	c.cancel = cancel
	c.mu.Unlock()

	syncInfo, err := c.connectOnce(ctx)
	if err != nil {
		cancel()
		return err
	}
	c.emitConnected(syncInfo)
	c.startUDP()
	c.mu.Lock()
	c.maintainStarted = true
	c.mu.Unlock()
	go c.maintain(ctx)
	return nil
}

// Close cleanly disconnects and stops reconnect.
func (c *Client) Close() error {
	c.closeOnce.Do(func() {
		close(c.closed)
		c.mu.RLock()
		cancel := c.cancel
		conn := c.conn
		maintainStarted := c.maintainStarted
		c.mu.RUnlock()
		if cancel != nil {
			cancel()
		}
		if conn != nil {
			_ = conn.Close()
		}
		c.stopUDP()
		// Disconnected is emitted by maintain when it observes the shutdown.
		// If maintain never started, clear local state here.
		if !maintainStarted {
			c.setConnected(false)
		}
	})
	c.mu.RLock()
	maintainStarted := c.maintainStarted
	c.mu.RUnlock()
	if maintainStarted {
		select {
		case <-c.maintainDone:
		case <-time.After(2 * time.Second):
		}
	}
	return nil
}

// Session returns the server-assigned session ID, or 0 if not connected.
func (c *Client) Session() uint32 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.session
}

// Roster returns the live channel/user registry.
func (c *Client) Roster() *Roster {
	return c.roster
}

// Crypt returns the UDP CryptState (may be ModeLite/Legacy/Secure after CryptSetup).
func (c *Client) Crypt() *crypto.CryptState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.crypt
}

// IsConnected reports whether a ServerSync session is active.
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// IsUDPActive reports whether UDP voice has been confirmed via ping echo.
func (c *Client) IsUDPActive() bool {
	c.mu.RLock()
	u := c.udp
	c.mu.RUnlock()
	if u == nil {
		return false
	}
	return u.IsActive()
}

// ChannelID returns the client's current channel, or 0 if unknown.
func (c *Client) ChannelID() uint32 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if u, ok := c.roster.UserBySession(c.session); ok {
		return u.ChannelID
	}
	return c.channelID
}

// IsTransmitting reports whether the client is currently gated for TX.
func (c *Client) IsTransmitting() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.transmitting
}

// StartTransmit opens the PTT gate (audio path sends while this is true).
func (c *Client) StartTransmit() {
	c.mu.Lock()
	c.transmitting = true
	c.mu.Unlock()
}

// StopTransmit closes the PTT gate and notes that a terminator should be sent
// by the audio path on the next opportunity.
func (c *Client) StopTransmit() {
	c.mu.Lock()
	c.transmitting = false
	c.mu.Unlock()
}

// NextVoiceSeq returns and increments the outbound voice sequence number.
func (c *Client) NextVoiceSeq() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	seq := c.voiceSeq
	c.voiceSeq++
	return seq
}

// JoinChannel moves the local user to the named channel.
func (c *Client) JoinChannel(name string) error {
	if !c.IsConnected() {
		return ErrNotConnected
	}
	ch, ok := c.roster.ChannelByName(name)
	if !ok {
		return fmt.Errorf("mumbleclient: channel %q not found", name)
	}
	return c.JoinChannelID(ch.ID)
}

// JoinChannelID moves the local user to channel id.
func (c *Client) JoinChannelID(id uint32) error {
	if !c.IsConnected() {
		return ErrNotConnected
	}
	c.mu.RLock()
	session := c.session
	c.mu.RUnlock()
	if err := c.WriteMessage(protocol.MessageUserState, &messages.UserState{
		Session:   session,
		ChannelID: id,
		SetFields: messages.UserStateSetSession | messages.UserStateSetChannelID,
	}); err != nil {
		return err
	}
	c.mu.Lock()
	c.channelID = id
	c.mu.Unlock()
	return nil
}

// On registers an event handler. Handlers run synchronously from client goroutines.
func (c *Client) On(handler EventHandler) {
	if handler == nil {
		return
	}
	c.handlersMu.Lock()
	c.handlers = append(c.handlers, handler)
	c.handlersMu.Unlock()
}

// Events returns a buffered event channel. Events are dropped if the buffer is full.
func (c *Client) Events() <-chan Event {
	return c.events
}

func (c *Client) emit(e Event) {
	c.handlersMu.RLock()
	handlers := append([]EventHandler(nil), c.handlers...)
	c.handlersMu.RUnlock()
	for _, h := range handlers {
		h(e)
	}
	select {
	case c.events <- e:
	default:
	}
}

func (c *Client) emitConnected(ss *messages.ServerSync) {
	ev := Connected{}
	if ss != nil {
		ev.Session = ss.Session
		ev.WelcomeText = ss.WelcomeText
		ev.MaxBandwidth = ss.MaxBandwidth
		ev.Permissions = ss.Permissions
	} else {
		ev.Session = c.Session()
	}
	c.emit(ev)
	c.emit(TransportChanged{Transport: TransportTCP})
}

func (c *Client) setConnected(v bool) {
	c.mu.Lock()
	c.connected = v
	if !v {
		c.session = 0
	}
	c.mu.Unlock()
}

func (c *Client) connectOnce(ctx context.Context) (*messages.ServerSync, error) {
	if c.cfg.Address == "" || c.cfg.Username == "" {
		return nil, errors.New("mumbleclient: address and username are required")
	}

	tlsCfg, err := c.tlsConfig()
	if err != nil {
		return nil, err
	}

	d := tls.Dialer{Config: tlsCfg}
	netConn, err := d.DialContext(ctx, "tcp", c.cfg.Address)
	if err != nil {
		return nil, fmt.Errorf("mumbleclient: dial: %w", err)
	}
	conn, ok := netConn.(*tls.Conn)
	if !ok {
		_ = netConn.Close()
		return nil, errors.New("mumbleclient: expected TLS connection")
	}

	peer, err := udpAddrFromTCP(conn.RemoteAddr())
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	c.mu.Lock()
	if c.conn != nil {
		_ = c.conn.Close()
	}
	c.conn = conn
	c.peerAddr = peer
	c.crypt = crypto.NewCryptState(crypto.ModeLegacy)
	c.session = 0
	c.connected = false
	c.mu.Unlock()
	c.roster.Reset()

	_ = conn.SetDeadline(time.Now().Add(TCPReadTimeout))
	syncInfo, err := c.handshake(conn)
	if err != nil {
		_ = conn.Close()
		c.mu.Lock()
		c.conn = nil
		c.mu.Unlock()
		return nil, err
	}
	_ = conn.SetDeadline(time.Time{})

	if err := c.joinConfiguredChannel(); err != nil {
		_ = conn.Close()
		c.mu.Lock()
		c.conn = nil
		c.connected = false
		c.mu.Unlock()
		return nil, err
	}

	c.setConnected(true)
	return syncInfo, nil
}

func (c *Client) tlsConfig() (*tls.Config, error) {
	cfg := &tls.Config{
		InsecureSkipVerify: c.cfg.InsecureTLS, //nolint:gosec // configurable for self-signed Murmur
		MinVersion:         tls.VersionTLS12,
	}
	if c.cfg.CertFile != "" || c.cfg.KeyFile != "" {
		if c.cfg.CertFile == "" || c.cfg.KeyFile == "" {
			return nil, errors.New("mumbleclient: both CertFile and KeyFile are required")
		}
		cert, err := tls.LoadX509KeyPair(c.cfg.CertFile, c.cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("mumbleclient: load cert: %w", err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}
	return cfg, nil
}

func (c *Client) handshake(conn *tls.Conn) (*messages.ServerSync, error) {
	msgType, payload, err := protocol.ReadPacket(conn)
	if err != nil {
		return nil, fmt.Errorf("mumbleclient: read server version: %w", err)
	}
	if msgType != protocol.MessageVersion {
		return nil, fmt.Errorf("mumbleclient: expected Version, got type %d", msgType)
	}
	_, _ = protocol.ReadMessage(msgType, payload)

	if err := c.writeMessageLocked(conn, protocol.MessageVersion, &messages.Version{
		Release:     c.cfg.Release,
		OS:          "Go",
		CryptoModes: c.cfg.CryptoModes,
	}); err != nil {
		return nil, fmt.Errorf("mumbleclient: send version: %w", err)
	}

	if err := c.writeMessageLocked(conn, protocol.MessageAuthenticate, &messages.Authenticate{
		Username: c.cfg.Username,
		Password: c.cfg.Password,
		Opus:     true,
	}); err != nil {
		return nil, fmt.Errorf("mumbleclient: send authenticate: %w", err)
	}

	for {
		select {
		case <-c.closed:
			return nil, errors.New("mumbleclient: closed during handshake")
		default:
		}
		_ = conn.SetDeadline(time.Now().Add(TCPReadTimeout))
		msgType, payload, err = protocol.ReadPacket(conn)
		if err != nil {
			return nil, fmt.Errorf("mumbleclient: handshake read: %w", err)
		}
		switch msgType {
		case protocol.MessageReject:
			msg, err := protocol.ReadMessage(msgType, payload)
			if err != nil {
				return nil, errors.New("mumbleclient: rejected")
			}
			if m, ok := msg.(*messages.Reject); ok {
				return nil, fmt.Errorf("mumbleclient: rejected: %s", m.Reason)
			}
			return nil, errors.New("mumbleclient: rejected")
		case protocol.MessageCryptSetup:
			if err := c.applyCryptSetup(payload); err != nil {
				return nil, err
			}
		case protocol.MessageChannelState:
			msg, err := protocol.ReadMessage(msgType, payload)
			if err != nil {
				continue
			}
			if cs, ok := msg.(*messages.ChannelState); ok {
				ch, added := c.roster.ApplyChannelState(cs)
				if added {
					c.emit(ChannelAdd{Channel: ch})
				} else {
					c.emit(ChannelUpdate{Channel: ch})
				}
			}
		case protocol.MessageUserState:
			msg, err := protocol.ReadMessage(msgType, payload)
			if err != nil {
				continue
			}
			if us, ok := msg.(*messages.UserState); ok {
				u, added := c.roster.ApplyUserState(us)
				if added {
					c.emit(UserAdd{User: u})
				} else {
					c.emit(UserUpdate{User: u})
				}
			}
		case protocol.MessageServerSync:
			msg, err := protocol.ReadMessage(msgType, payload)
			if err != nil {
				return nil, fmt.Errorf("mumbleclient: server sync: %w", err)
			}
			ss, ok := msg.(*messages.ServerSync)
			if !ok || ss == nil {
				return nil, errors.New("mumbleclient: invalid server sync")
			}
			c.mu.Lock()
			c.session = ss.Session
			c.mu.Unlock()
			return ss, nil
		case protocol.MessageCodecVersion, protocol.MessageServerConfig, protocol.MessageVersion:
			// ignore
		default:
			// ignore other pre-sync messages
		}
	}
}

func (c *Client) joinConfiguredChannel() error {
	if c.cfg.Channel == "" {
		return nil
	}
	ch, ok := c.roster.ChannelByName(c.cfg.Channel)
	if !ok {
		return fmt.Errorf("mumbleclient: channel %q not found", c.cfg.Channel)
	}
	c.mu.RLock()
	session := c.session
	c.mu.RUnlock()
	return c.WriteMessage(protocol.MessageUserState, &messages.UserState{
		Session:   session,
		ChannelID: ch.ID,
		SetFields: messages.UserStateSetSession | messages.UserStateSetChannelID,
	})
}

func (c *Client) applyCryptSetup(payload []byte) error {
	msg, err := protocol.ReadMessage(protocol.MessageCryptSetup, payload)
	if err != nil {
		return fmt.Errorf("mumbleclient: crypt setup parse: %w", err)
	}
	cs, ok := msg.(*messages.CryptSetup)
	if !ok || cs == nil {
		return errors.New("mumbleclient: invalid crypt setup")
	}
	// Key present (including empty lite key) or both nonces => full setup.
	// Key absent with a lone ServerNonce => decrypt-nonce resync.
	if cs.Key != nil || (len(cs.ClientNonce) > 0 && len(cs.ServerNonce) > 0) {
		mode := cryptModeFromKey(cs.Key)
		state := crypto.NewCryptState(mode)
		if err := state.SetKey(cs.Key, cs.ClientNonce, cs.ServerNonce); err != nil {
			return fmt.Errorf("mumbleclient: crypt setkey: %w", err)
		}
		c.mu.Lock()
		c.crypt = state
		c.mu.Unlock()
		return nil
	}
	c.mu.RLock()
	crypt := c.crypt
	c.mu.RUnlock()
	if crypt == nil {
		return nil
	}
	if len(cs.ServerNonce) > 0 {
		_ = crypt.SetDecNonce(cs.ServerNonce)
	}
	if len(cs.ClientNonce) == 0 && len(cs.Key) == 0 {
		_ = c.WriteMessage(protocol.MessageCryptSetup, &messages.CryptSetup{
			ClientNonce: crypt.EncNonce(),
		})
	}
	return nil
}

// WriteMessage marshals and writes a control message on the TLS connection.
func (c *Client) WriteMessage(msgType protocol.MessageType, msg protocol.Message) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	if conn == nil {
		return ErrNotConnected
	}
	return c.writeMessageLocked(conn, msgType, msg)
}

func (c *Client) writeMessageLocked(conn *tls.Conn, msgType protocol.MessageType, msg protocol.Message) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return protocol.WriteMessage(conn, msgType, msg)
}

// WritePacket writes a raw typed control packet (e.g. UDPTunnel voice).
func (c *Client) WritePacket(msgType protocol.MessageType, payload []byte) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	if conn == nil {
		return ErrNotConnected
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return protocol.WritePacket(conn, msgType, payload)
}

func udpAddrFromTCP(addr net.Addr) (*net.UDPAddr, error) {
	if tcp, ok := addr.(*net.TCPAddr); ok {
		return &net.UDPAddr{IP: append(net.IP(nil), tcp.IP...), Port: tcp.Port}, nil
	}
	host, portStr, err := net.SplitHostPort(addr.String())
	if err != nil {
		return nil, fmt.Errorf("mumbleclient: peer addr: %w", err)
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return nil, fmt.Errorf("mumbleclient: peer IP %q not numeric", host)
	}
	port, err := net.LookupPort("udp", portStr)
	if err != nil {
		return nil, fmt.Errorf("mumbleclient: peer port: %w", err)
	}
	return &net.UDPAddr{IP: ip, Port: port}, nil
}

func cryptModeFromKey(key []byte) crypto.Mode {
	switch len(key) {
	case 32:
		return crypto.ModeSecure
	case 0:
		return crypto.ModeLite
	default:
		return crypto.ModeLegacy
	}
}

// nextBackoff doubles delay, clamped to [ReconnectMinDelay, ReconnectMaxDelay].
func nextBackoff(d time.Duration) time.Duration {
	if d < ReconnectMinDelay {
		return ReconnectMinDelay
	}
	next := d * 2
	if next > ReconnectMaxDelay {
		return ReconnectMaxDelay
	}
	return next
}
