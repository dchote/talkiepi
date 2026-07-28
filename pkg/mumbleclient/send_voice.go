package mumbleclient

import (
	"github.com/dchote/go-mumble-server/pkg/mumble/audio"
	"github.com/dchote/go-mumble-server/pkg/mumble/protocol"
)

// SendVoice sends one Opus voice frame (or a talk-spurt terminator) to the server.
// Prefers UDP when active; falls back to TCP UDPTunnel.
func (c *Client) SendVoice(payload []byte, terminator bool) error {
	if !c.IsConnected() {
		return ErrNotConnected
	}
	seq := c.NextVoiceSeq()
	buf := make([]byte, 1+audio.MaxVarintLen*2+len(payload)+8)
	n, err := BuildClientVoice(buf, audio.CodecOpus, 0, seq, payload, terminator)
	if err != nil {
		return err
	}
	packet := buf[:n]
	if c.IsUDPActive() {
		if err := c.SendUDP(packet); err == nil {
			return nil
		}
		// Fall through to TCP tunnel on UDP send failure.
	}
	return c.WritePacket(protocol.MessageUDPTunnel, packet)
}
