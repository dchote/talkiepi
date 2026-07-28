//go:build cgo && opus

package codec

import (
	"fmt"

	"gopkg.in/hraban/opus.v2"
)

// Decoder wraps an Opus decoder for a Profile.
type Decoder struct {
	dec     *opus.Decoder
	profile Profile
	pcm     []int16
}

// NewDecoder creates an Opus decoder for the profile (requires CGO + libopus).
func NewDecoder(p Profile) (*Decoder, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	dec, err := opus.NewDecoder(p.SampleRate, p.Channels)
	if err != nil {
		return nil, fmt.Errorf("codec: opus decoder: %w", err)
	}
	frame := p.FrameSamples() * p.Channels
	return &Decoder{
		dec:     dec,
		profile: p,
		pcm:     make([]int16, frame),
	}, nil
}

// Decode decodes one Opus packet into PCM. Returns a copy of frame samples.
func (d *Decoder) Decode(payload []byte) ([]int16, error) {
	if len(payload) == 0 {
		return nil, fmt.Errorf("codec: empty opus payload")
	}
	n, err := d.dec.Decode(payload, d.pcm)
	if err != nil {
		return nil, err
	}
	need := d.profile.FrameSamples() * d.profile.Channels
	out := make([]int16, need)
	copy(out, d.pcm[:n*d.profile.Channels])
	if n*d.profile.Channels < need {
		// Pad short frames (rare) with silence.
		for i := n * d.profile.Channels; i < need; i++ {
			out[i] = 0
		}
	}
	return out, nil
}

// DecodePLC synthesizes a concealment frame after a lost packet.
func (d *Decoder) DecodePLC() ([]int16, error) {
	need := d.profile.FrameSamples() * d.profile.Channels
	pcm := make([]int16, need)
	if err := d.dec.DecodePLC(pcm); err != nil {
		return nil, err
	}
	return pcm, nil
}

// Close clears the decoder handle.
func (d *Decoder) Close() error {
	d.dec = nil
	return nil
}
