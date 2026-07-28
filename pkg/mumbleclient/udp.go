package mumbleclient

import (
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dchote/go-mumble-server/pkg/mumble/crypto"
)

const maxUDPPacket = 1024

// udpTransport handles encrypted UDP voice and connectivity probes.
type udpTransport struct {
	client *Client

	conn     *net.UDPConn
	peer     *net.UDPAddr
	mu       sync.Mutex
	active   atomic.Bool
	stopped  chan struct{}
	stopOnce sync.Once

	initialSent int
}

func (c *Client) startUDP() {
	c.stopUDP()

	c.mu.RLock()
	peer := c.peerAddr
	c.mu.RUnlock()
	if peer == nil {
		return
	}

	conn, err := net.DialUDP("udp", nil, peer)
	if err != nil {
		return
	}
	u := &udpTransport{
		client:  c,
		conn:    conn,
		peer:    peer,
		stopped: make(chan struct{}),
	}
	c.mu.Lock()
	c.udp = u
	c.mu.Unlock()
	go u.loop()
}

func (c *Client) stopUDP() {
	c.mu.Lock()
	u := c.udp
	c.udp = nil
	c.mu.Unlock()
	if u != nil {
		wasActive := u.IsActive()
		u.stop()
		if wasActive {
			c.emit(TransportChanged{Transport: TransportTCP})
		}
	}
}

func (u *udpTransport) IsActive() bool {
	return u.active.Load()
}

func (u *udpTransport) stop() {
	u.stopOnce.Do(func() {
		close(u.stopped)
		u.mu.Lock()
		if u.conn != nil {
			_ = u.conn.Close()
			u.conn = nil
		}
		u.mu.Unlock()
		u.active.Store(false)
	})
}

func (u *udpTransport) loop() {
	go u.readLoop()

	ticker := time.NewTimer(0) // send first ping immediately
	defer ticker.Stop()

	for {
		select {
		case <-u.stopped:
			return
		case <-ticker.C:
			u.sendPing()
			u.mu.Lock()
			n := u.initialSent
			u.mu.Unlock()
			interval := UDPPingInterval
			if n < UDPPingInitialCount {
				interval = UDPPingInitialInterval
			}
			ticker.Reset(interval)
		}
	}
}

func (u *udpTransport) readLoop() {
	buf := make([]byte, maxUDPPacket+64)
	for {
		u.mu.Lock()
		conn := u.conn
		u.mu.Unlock()
		if conn == nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(UDPPingInterval * 2))
		n, err := conn.Read(buf)
		if err != nil {
			select {
			case <-u.stopped:
				return
			default:
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					continue
				}
				return
			}
		}
		u.handlePacket(buf[:n])
	}
}

func (u *udpTransport) handlePacket(data []byte) {
	plain, err := u.Decrypt(data)
	if err != nil || len(plain) == 0 {
		return
	}
	if _, ok := ParseUDPPing(plain); ok {
		if !u.active.Swap(true) {
			u.client.emit(TransportChanged{Transport: TransportUDP})
		}
		return
	}
	u.client.emit(VoicePacket{
		Data:   append([]byte(nil), plain...),
		ViaUDP: true,
	})
}

func (u *udpTransport) sendPing() {
	buf := make([]byte, 16)
	n, err := BuildUDPPing(buf, time.Now().UnixMilli())
	if err != nil {
		return
	}
	if err := u.Send(buf[:n]); err != nil {
		return
	}
	u.mu.Lock()
	if u.initialSent < UDPPingInitialCount {
		u.initialSent++
	}
	u.mu.Unlock()
}

// Send encrypts and writes a UDP datagram to the TLS peer address.
func (u *udpTransport) Send(plain []byte) error {
	enc, err := u.Encrypt(plain)
	if err != nil {
		return err
	}
	u.mu.Lock()
	conn := u.conn
	u.mu.Unlock()
	if conn == nil {
		return net.ErrClosed
	}
	_, err = conn.Write(enc)
	return err
}

// Encrypt seals plain with the client's CryptState.
func (u *udpTransport) Encrypt(plain []byte) ([]byte, error) {
	crypt := u.crypt()
	if crypt == nil {
		out := make([]byte, len(plain))
		copy(out, plain)
		return out, nil
	}
	out := make([]byte, len(plain)+crypt.Overhead())
	if err := crypt.Encrypt(out, plain); err != nil {
		return nil, err
	}
	return out, nil
}

// Decrypt opens a UDP datagram with the client's CryptState.
func (u *udpTransport) Decrypt(packet []byte) ([]byte, error) {
	crypt := u.crypt()
	if crypt == nil {
		out := make([]byte, len(packet))
		copy(out, packet)
		return out, nil
	}
	oh := crypt.Overhead()
	if len(packet) < oh {
		return nil, crypto.ErrInvalidPacket
	}
	out := make([]byte, len(packet)-oh)
	if err := crypt.Decrypt(out, packet); err != nil {
		return nil, err
	}
	return out, nil
}

func (u *udpTransport) crypt() *crypto.CryptState {
	return u.client.Crypt()
}

// SendUDP writes an encrypted voice/ping packet on the UDP transport.
// Returns false if UDP is not started.
func (c *Client) SendUDP(plain []byte) error {
	c.mu.RLock()
	u := c.udp
	c.mu.RUnlock()
	if u == nil {
		return ErrNotConnected
	}
	return u.Send(plain)
}
