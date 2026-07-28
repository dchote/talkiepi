//go:build !cgo || !opus

package codec

import "errors"

// Encoder is a stub when the opus CGO build tag is not enabled.
type Encoder struct {
	profile Profile
}

// NewEncoder returns a stub encoder that rejects Encode calls.
func NewEncoder(p Profile) (*Encoder, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return &Encoder{profile: p}, nil
}

// Encode always fails without libopus linked via -tags opus.
func (e *Encoder) Encode(pcm []int16) ([]byte, error) {
	return nil, errors.New("opus not available")
}

// Close is a no-op for the stub.
func (e *Encoder) Close() error { return nil }
