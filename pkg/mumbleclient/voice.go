package mumbleclient

import (
	"errors"

	"github.com/dchote/go-mumble-server/pkg/mumble/audio"
)

// Voice packet errors.
var (
	ErrVoiceTooShort   = errors.New("mumbleclient: voice packet too short")
	ErrVoiceTruncated  = errors.New("mumbleclient: voice packet truncated")
	ErrVoiceEncodeSize = errors.New("mumbleclient: voice encode buffer too small")
)

const (
	// TerminatorBit is set in the payload-length varint to mark end of a talk spurt.
	TerminatorBit = 0x2000
	// PayloadLenMask masks out the terminator bit from a payload-length varint.
	PayloadLenMask = 0x1FFF
)

// ClientVoice is a client-to-server legacy binary voice packet (no session field).
type ClientVoice struct {
	Codec        uint8
	Target       uint8
	Sequence     int64
	IsTerminator bool
	Payload      []byte
}

// ServerVoice is a server-to-client legacy binary voice packet (includes session).
type ServerVoice struct {
	Codec        uint8
	Target       uint8
	Session      uint32
	Sequence     int64
	IsTerminator bool
	Payload      []byte
}

// BuildClientVoice encodes a client-to-server Opus voice packet into dst.
// Returns the number of bytes written.
func BuildClientVoice(dst []byte, codec, target uint8, sequence int64, payload []byte, terminator bool) (int, error) {
	need := 1 + audio.MaxVarintLen*2 + len(payload)
	if len(dst) < need {
		return 0, ErrVoiceEncodeSize
	}
	dst[0] = (codec << 5) | (target & 0x1F)
	n := 1
	n += audio.EncodeVarint(dst[n:], sequence)

	lenVal := int64(len(payload))
	if terminator {
		lenVal |= TerminatorBit
	}
	n += audio.EncodeVarint(dst[n:], lenVal)
	if n+len(payload) > len(dst) {
		return 0, ErrVoiceEncodeSize
	}
	copy(dst[n:], payload)
	return n + len(payload), nil
}

// ParseClientVoice parses a client-to-server legacy voice packet.
// Round-trips with BuildClientVoice and audio.ParsePacket.
func ParseClientVoice(data []byte) (*ClientVoice, error) {
	p, err := audio.ParsePacket(data)
	if err != nil {
		return nil, err
	}
	return &ClientVoice{
		Codec:        p.Codec,
		Target:       p.Target,
		Sequence:     p.Sequence,
		IsTerminator: p.IsTerminator,
		Payload:      append([]byte(nil), p.Payload...),
	}, nil
}

// ParseServerVoice parses a server-to-client legacy voice packet.
// Format: header | session varint | sequence varint | length varint | payload.
func ParseServerVoice(data []byte) (*ServerVoice, error) {
	if len(data) < 2 {
		return nil, ErrVoiceTooShort
	}
	v := &ServerVoice{
		Codec:  (data[0] >> 5) & 0x07,
		Target: data[0] & 0x1F,
	}
	buf := data[1:]

	session, n := audio.DecodeVarint(buf)
	if n <= 0 {
		return nil, ErrVoiceTooShort
	}
	v.Session = uint32(session)
	buf = buf[n:]

	seq, n := audio.DecodeVarint(buf)
	if n <= 0 {
		return nil, ErrVoiceTooShort
	}
	v.Sequence = seq
	buf = buf[n:]

	rawLen, n := audio.DecodeVarint(buf)
	if n <= 0 {
		return nil, ErrVoiceTooShort
	}
	v.IsTerminator = (rawLen & TerminatorBit) != 0
	payloadLen := int(rawLen & PayloadLenMask)
	buf = buf[n:]

	if len(buf) < payloadLen {
		return nil, ErrVoiceTruncated
	}
	v.Payload = append([]byte(nil), buf[:payloadLen]...)
	return v, nil
}

// BuildUDPPing builds a UDP ping packet (codec=Ping header 0x20 + timestamp varint).
func BuildUDPPing(dst []byte, timestamp int64) (int, error) {
	if len(dst) < 1+audio.MaxVarintLen {
		return 0, ErrVoiceEncodeSize
	}
	dst[0] = 0x20 // codec ping
	n := 1 + audio.EncodeVarint(dst[1:], timestamp)
	return n, nil
}

// ParseUDPPing parses a UDP ping echo. Returns timestamp and true if this is a ping.
func ParseUDPPing(data []byte) (timestamp int64, ok bool) {
	if len(data) < 2 {
		return 0, false
	}
	if (data[0]>>5)&0x07 != 1 { // CodecPing = 1
		return 0, false
	}
	ts, n := audio.DecodeVarint(data[1:])
	if n <= 0 {
		return 0, false
	}
	return ts, true
}
