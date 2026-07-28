//go:build cgo && opus

package codec

import (
	"fmt"

	"gopkg.in/hraban/opus.v2"
)

// Encoder wraps an Opus encoder for a Profile.
type Encoder struct {
	enc     *opus.Encoder
	profile Profile
	buf     []byte
}

// NewEncoder creates an Opus encoder for the profile (requires CGO + libopus).
func NewEncoder(p Profile) (*Encoder, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	enc, err := opus.NewEncoder(p.SampleRate, p.Channels, opus.AppVoIP)
	if err != nil {
		return nil, fmt.Errorf("codec: opus encoder: %w", err)
	}
	if err := enc.SetBitrate(p.Bitrate); err != nil {
		return nil, err
	}
	_ = enc.SetComplexity(p.Complexity)
	return &Encoder{
		enc:     enc,
		profile: p,
		buf:     make([]byte, 4000),
	}, nil
}

// Encode encodes one PCM frame into Opus. pcm length must be FrameSamples*Channels.
func (e *Encoder) Encode(pcm []int16) ([]byte, error) {
	need := e.profile.FrameSamples() * e.profile.Channels
	if len(pcm) != need {
		return nil, fmt.Errorf("codec: expected %d samples, got %d", need, len(pcm))
	}
	n, err := e.enc.Encode(pcm, e.buf)
	if err != nil {
		return nil, err
	}
	out := make([]byte, n)
	copy(out, e.buf[:n])
	return out, nil
}

// Close clears the encoder handle.
func (e *Encoder) Close() error {
	e.enc = nil
	return nil
}
