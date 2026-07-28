package audio

import (
	"context"
	"sync"

	"github.com/dchote/talkiepi/internal/audio/process"
	"github.com/dchote/talkiepi/internal/codec"
	"github.com/dchote/talkiepi/internal/vad"
)

// VoiceSink accepts encoded Opus frames from the capture path.
type VoiceSink interface {
	SendVoice(payload []byte, terminator bool) error
}

// SessionConfig configures a capture/playback session.
type SessionConfig struct {
	Profile    codec.Profile
	Processing process.Config
	Input      Descriptor
	Output     Descriptor
}

// Session owns the capture → process → VAD → encode path and the
// decode → jitter → mix → playback path. Device I/O is provided by
// CaptureSource / PlaybackSink implementations (malgo when built with -tags malgo).
type Session struct {
	cfg       SessionConfig
	processor process.Processor
	vad       *vad.Detector
	capture   *Ring
	playback  *Ring
	mixer     *Mixer
	sink      VoiceSink

	mu     sync.Mutex
	cancel context.CancelFunc
}

// NewSession builds a session with rings, mixer, VAD, and processor.
func NewSession(cfg SessionConfig, sink VoiceSink) (*Session, error) {
	if err := cfg.Profile.Validate(); err != nil {
		return nil, err
	}
	proc, err := process.New(cfg.Processing)
	if err != nil {
		return nil, err
	}
	frame := cfg.Profile.FrameSamples()
	return &Session{
		cfg:       cfg,
		processor: proc,
		vad:       vad.New(vad.DefaultConfig()),
		capture:   NewRing(frame * 50),
		playback:  NewRing(frame * 50),
		mixer:     NewMixer(frame),
		sink:      sink,
	}, nil
}

// ProcessCaptureFrame runs one mic frame through AEC/VAD and optionally sends Opus.
// encode may be nil (tests); when non-nil it must return Opus bytes.
func (s *Session) ProcessCaptureFrame(pcm []int16, encode func([]int16) ([]byte, error), allowTX bool) error {
	cleaned, err := s.processor.ProcessCapture(pcm)
	if err != nil {
		return err
	}
	// Drain any ForceStop terminator before Process (which clears the flag).
	forcedTerm := s.vad.TakeTerminator()
	res := s.vad.Process(cleaned)
	if (res.SendTerminator || forcedTerm) && s.sink != nil {
		return s.sink.SendVoice(nil, true)
	}
	if !allowTX || !res.Transmit || encode == nil || s.sink == nil {
		return nil
	}
	payload, err := encode(cleaned)
	if err != nil {
		return err
	}
	return s.sink.SendVoice(payload, false)
}

// FeedFarEnd provides AEC reference (pre-device mix).
func (s *Session) FeedFarEnd(pcm []int16) error {
	return s.processor.ProcessRender(pcm)
}

// CaptureRing exposes the capture ring for device callbacks.
func (s *Session) CaptureRing() *Ring { return s.capture }

// PlaybackRing exposes the playback ring for device callbacks.
func (s *Session) PlaybackRing() *Ring { return s.playback }

// Mixer returns the playback mixer.
func (s *Session) Mixer() *Mixer { return s.mixer }

// ProcessorStats returns AEC stats.
func (s *Session) ProcessorStats() process.Stats { return s.processor.Stats() }

// Close releases the processor.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	return s.processor.Close()
}
