package audio

import (
	"testing"

	"github.com/dchote/talkiepi/internal/audio/process"
	"github.com/dchote/talkiepi/internal/codec"
)

type sinkBuf struct {
	frames int
	terms  int
}

func (s *sinkBuf) SendVoice(payload []byte, terminator bool) error {
	if terminator {
		s.terms++
		return nil
	}
	s.frames++
	return nil
}

func TestSessionProcessCaptureGate(t *testing.T) {
	sink := &sinkBuf{}
	s, err := NewSession(SessionConfig{
		Profile:    codec.Default(),
		Processing: process.DefaultConfig(48000, 960),
	}, sink)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	pcm := make([]int16, 960)
	for i := range pcm {
		pcm[i] = 5000
	}
	enc := func([]int16) ([]byte, error) { return []byte{1, 2, 3}, nil }

	// Not allowed to TX
	if err := s.ProcessCaptureFrame(pcm, enc, false); err != nil {
		t.Fatal(err)
	}
	if sink.frames != 0 {
		t.Fatal("should not send when allowTX=false")
	}

	// Force VAD into transmit then allow
	s.vad.ForceTransmit()
	if err := s.ProcessCaptureFrame(pcm, enc, true); err != nil {
		t.Fatal(err)
	}
	if sink.frames != 1 {
		t.Fatalf("frames=%d", sink.frames)
	}

	s.vad.ForceStop()
	if err := s.ProcessCaptureFrame(pcm, enc, true); err != nil {
		t.Fatal(err)
	}
	if sink.terms < 1 {
		t.Fatal("expected terminator")
	}
}
