package process

import (
	"fmt"
	"strings"
)

// Backend identifies a processing implementation.
type Backend string

const (
	BackendNone   Backend = "none"
	BackendSpeex  Backend = "speex"
	BackendWebRTC Backend = "webrtc"
)

// Config configures the audio processor.
type Config struct {
	Backend       Backend
	SampleRate    int
	FrameSamples  int
	EchoCancel    bool
	EchoTailMs    int
	EchoDelayMs   int
	NoiseSuppress bool
	NSLevel       string
	AGC           bool
	AGCTargetDBFS float64
	Dereverb      bool
	HighPass      bool
}

// DefaultConfig returns Speex-oriented defaults.
func DefaultConfig(sampleRate, frameSamples int) Config {
	return Config{
		Backend:       BackendSpeex,
		SampleRate:    sampleRate,
		FrameSamples:  frameSamples,
		EchoCancel:    true,
		EchoTailMs:    150,
		NoiseSuppress: true,
		NSLevel:       "moderate",
		AGCTargetDBFS: -18,
		HighPass:      true,
	}
}

// Stats exposes runtime processing metrics.
type Stats struct {
	Backend string  `json:"backend"`
	ERLE    float64 `json:"erle"` // echo return loss enhancement estimate (dB)
	Active  bool    `json:"active"`
}

// Processor removes echo / noise from capture using a far-end reference.
type Processor interface {
	// ProcessRender feeds far-end (playback) samples.
	ProcessRender(far []int16) error
	// ProcessCapture processes near-end (mic) samples in place and returns cleaned audio.
	ProcessCapture(near []int16) ([]int16, error)
	Stats() Stats
	Close() error
}

// New creates a Processor for cfg. Unknown backends fall back to none.
func New(cfg Config) (Processor, error) {
	if cfg.SampleRate <= 0 {
		cfg.SampleRate = 48000
	}
	if cfg.FrameSamples <= 0 {
		cfg.FrameSamples = cfg.SampleRate / 50
	}
	switch Backend(strings.ToLower(string(cfg.Backend))) {
	case BackendNone, "":
		return NewNone(), nil
	case BackendSpeex:
		return newSpeex(cfg)
	case BackendWebRTC:
		return newWebRTC(cfg)
	default:
		return nil, fmt.Errorf("process: unknown backend %q", cfg.Backend)
	}
}

// noneProcessor passes audio through unchanged.
type noneProcessor struct{}

// NewNone returns a no-op processor.
func NewNone() Processor { return noneProcessor{} }

func (noneProcessor) ProcessRender([]int16) error { return nil }
func (noneProcessor) ProcessCapture(near []int16) ([]int16, error) {
	out := make([]int16, len(near))
	copy(out, near)
	return out, nil
}
func (noneProcessor) Stats() Stats { return Stats{Backend: string(BackendNone)} }
func (noneProcessor) Close() error { return nil }
