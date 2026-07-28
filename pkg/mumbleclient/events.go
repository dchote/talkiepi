package mumbleclient

import (
	"time"

	"github.com/dchote/go-mumble-server/pkg/mumble/protocol/messages"
)

// Event is a client lifecycle or server-state notification.
type Event interface {
	event()
}

// EventHandler receives client events.
type EventHandler func(Event)

// Transport identifies the active voice path.
type Transport int

const (
	// TransportTCP means voice is carried in UDPTunnel over TLS.
	TransportTCP Transport = iota
	// TransportUDP means encrypted UDP voice is confirmed active.
	TransportUDP
)

func (t Transport) String() string {
	switch t {
	case TransportUDP:
		return "udp"
	default:
		return "tcp"
	}
}

// Connected is emitted after ServerSync (and optional channel join) succeed.
type Connected struct {
	Session      uint32
	WelcomeText  string
	MaxBandwidth uint32
	Permissions  uint64
}

func (Connected) event() {}

// Disconnected is emitted when an established session ends.
// Err is nil for a clean Close.
type Disconnected struct {
	Err error
}

func (Disconnected) event() {}

// UserAdd is emitted when a new user appears in the roster.
type UserAdd struct {
	User User
}

func (UserAdd) event() {}

// UserUpdate is emitted when an existing user changes state.
type UserUpdate struct {
	User User
}

func (UserUpdate) event() {}

// UserRemove is emitted when a user leaves or is removed.
type UserRemove struct {
	Session uint32
	Actor   uint32
	Reason  string
	Ban     bool
	User    User // snapshot before removal; zero if unknown
}

func (UserRemove) event() {}

// ChannelAdd is emitted when a new channel appears.
type ChannelAdd struct {
	Channel Channel
}

func (ChannelAdd) event() {}

// ChannelUpdate is emitted when an existing channel changes.
type ChannelUpdate struct {
	Channel Channel
}

func (ChannelUpdate) event() {}

// ChannelRemove is emitted when a channel is deleted.
type ChannelRemove struct {
	ChannelID uint32
	Channel   Channel // snapshot before removal; zero if unknown
}

func (ChannelRemove) event() {}

// TextMessage is an inbound text message.
type TextMessage struct {
	Actor      uint32
	Sessions   []uint32
	ChannelIDs []uint32
	TreeIDs    []uint32
	Message    string
}

func (TextMessage) event() {}

// PermissionDenied is an inbound permission denial.
type PermissionDenied struct {
	Permission uint32
	ChannelID  uint32
	Session    uint32
	Reason     string
	Type       messages.DenyType
	Name       string
}

func (PermissionDenied) event() {}

// VoicePacket is inbound voice (UDP or TCP tunnel), already decrypted if UDP.
type VoicePacket struct {
	Data   []byte
	ViaUDP bool
}

func (VoicePacket) event() {}

// PingRTT reports the round-trip time of a TCP Ping exchange.
type PingRTT struct {
	RTT time.Duration
}

func (PingRTT) event() {}

// TransportChanged reports a switch between UDP and TCP voice paths.
type TransportChanged struct {
	Transport Transport
}

func (TransportChanged) event() {}
