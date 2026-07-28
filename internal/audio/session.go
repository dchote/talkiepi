package audio

import (
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
	// InputGain is a linear amplitude scale applied after processing (from gain_db).
	InputGain float64
}

// Session owns capture → process → VAD → encode helpers and playback rings/mixer.
// Device I/O is provided by DeviceIO (malgo when built with CGO).
type Session struct {
	cfg       SessionConfig
	processor process.Processor
	vad       *vad.Detector
	capture   *Ring
	playback  *Ring
	mixer     *Mixer
	sink      VoiceSink
	inputGain float64

	mu sync.Mutex
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
	inGain := cfg.InputGain
	if inGain <= 0 {
		inGain = 1
	}
	return &Session{
		cfg:       cfg,
		processor: proc,
		vad:       vad.New(vad.DefaultConfig()),
		capture:   NewRing(frame * 50),
		playback:  NewRing(frame * 50),
		mixer:     NewMixer(frame),
		sink:      sink,
		inputGain: inGain,
	}, nil
}

// ProcessCaptureFrame runs one mic frame through AEC/VAD and optionally sends Opus.
// When allowTX is true, the VAD transmit gate is held open (PTT / always-on /
// communicator open) so hangover does not end TX while the mode still allows it.
// encode may be nil (tests); when non-nil it must return Opus bytes.
// speech reports whether the VAD classified the frame as speech.
func (s *Session) ProcessCaptureFrame(pcm []int16, encode func([]int16) ([]byte, error), allowTX bool) (speech bool, err error) {
	cleaned, err := s.processor.ProcessCapture(pcm)
	if err != nil {
		return false, err
	}
	ApplyLinearGain(cleaned, s.inputGain)

	// Deliver a pending terminator (e.g. PTT release) before re-opening the gate.
	if forcedTerm := s.vad.TakeTerminator(); forcedTerm && s.sink != nil {
		res := s.vad.Process(cleaned)
		return res.Speech, s.sink.SendVoice(nil, true)
	}

	// Hold the gate open for the whole allowed TX window (industry PTT practice).
	if allowTX {
		s.vad.ForceTransmit()
	}

	res := s.vad.Process(cleaned)
	if res.SendTerminator && s.sink != nil {
		return res.Speech, s.sink.SendVoice(nil, true)
	}
	if !allowTX || !res.Transmit || encode == nil || s.sink == nil {
		return res.Speech, nil
	}
	payload, err := encode(cleaned)
	if err != nil {
		return res.Speech, err
	}
	return res.Speech, s.sink.SendVoice(payload, false)
}

// ForceTransmit opens the VAD transmit gate (PTT / always-on assist).
func (s *Session) ForceTransmit() { s.vad.ForceTransmit() }

// ForceStop closes the VAD transmit gate and queues a terminator.
func (s *Session) ForceStop() { s.vad.ForceStop() }

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
	return s.processor.Close()
}
