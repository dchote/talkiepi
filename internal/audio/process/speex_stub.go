//go:build !cgo || !speexdsp

package process

import (
	"math"
)

// Stub Speex processor used when SpeexDSP is not linked.
// Implements a simple delay-based echo subtractor for tests and platforms
// without libspeexdsp. Not production quality — enable -tags speexdsp for real AEC.
type speexStub struct {
	cfg     Config
	farRing []int16
	pos     int
	erle    float64
}

func newSpeex(cfg Config) (Processor, error) {
	delay := cfg.EchoDelayMs * cfg.SampleRate / 1000
	if delay < 0 {
		delay = 0
	}
	tail := cfg.EchoTailMs * cfg.SampleRate / 1000
	if tail < cfg.FrameSamples {
		tail = cfg.FrameSamples * 8
	}
	size := delay + tail
	if size < cfg.FrameSamples {
		size = cfg.FrameSamples
	}
	return &speexStub{
		cfg:     cfg,
		farRing: make([]int16, size),
	}, nil
}

func (p *speexStub) ProcessRender(far []int16) error {
	for _, s := range far {
		p.farRing[p.pos] = s
		p.pos = (p.pos + 1) % len(p.farRing)
	}
	return nil
}

func (p *speexStub) ProcessCapture(near []int16) ([]int16, error) {
	out := make([]int16, len(near))
	if !p.cfg.EchoCancel || len(p.farRing) == 0 {
		copy(out, near)
		return out, nil
	}
	delay := p.cfg.EchoDelayMs * p.cfg.SampleRate / 1000
	var nearPow, errPow float64
	for i, s := range near {
		// Align with the most recently written far-end sample, offset by delay.
		idx := p.pos - 1 - delay
		for idx < 0 {
			idx += len(p.farRing)
		}
		idx = (idx + i) % len(p.farRing)
		// Test double: subtract a scaled delayed far-end reference.
		ref := int32(p.farRing[idx]) / 2
		v := int32(s) - ref
		if v > math.MaxInt16 {
			v = math.MaxInt16
		}
		if v < math.MinInt16 {
			v = math.MinInt16
		}
		out[i] = int16(v)
		nearPow += float64(s) * float64(s)
		errPow += float64(out[i]) * float64(out[i])
	}
	if errPow < 1 {
		errPow = 1
	}
	if nearPow > errPow {
		p.erle = 10 * math.Log10(nearPow/errPow)
	}
	return out, nil
}

func (p *speexStub) Stats() Stats {
	return Stats{Backend: string(BackendSpeex) + "_stub", ERLE: p.erle, Active: p.cfg.EchoCancel}
}

func (p *speexStub) Close() error { return nil }
