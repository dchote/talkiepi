//go:build !cgo || !opus

package codec

import "errors"

// Decoder is a stub when the opus CGO build tag is not enabled.
type Decoder struct {
	profile Profile
}

// NewDecoder returns a stub decoder.
func NewDecoder(p Profile) (*Decoder, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return &Decoder{profile: p}, nil
}

// Decode always fails without libopus linked via -tags opus.
func (d *Decoder) Decode(payload []byte) ([]int16, error) {
	return nil, errors.New("opus not available")
}

// DecodePLC always fails without libopus.
func (d *Decoder) DecodePLC() ([]int16, error) {
	return nil, errors.New("opus not available")
}

// Close is a no-op for the stub.
func (d *Decoder) Close() error { return nil }
