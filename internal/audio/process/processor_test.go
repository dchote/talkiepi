package process

import (
	"math"
	"testing"
)

func TestNonePassthrough(t *testing.T) {
	p := NewNone()
	in := []int16{1, 2, 3}
	out, err := p.ProcessCapture(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 || out[0] != 1 {
		t.Fatalf("out=%v", out)
	}
	_ = p.ProcessRender(in)
	if p.Stats().Backend != "none" {
		t.Fatal(p.Stats())
	}
}

func TestNewSpeexStub(t *testing.T) {
	p, err := New(DefaultConfig(16000, 320))
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if p.Stats().Backend == "" {
		t.Fatal("empty backend")
	}
}

func TestAECImprovesERLE(t *testing.T) {
	cfg := DefaultConfig(16000, 320)
	cfg.EchoDelayMs = 10
	cfg.EchoTailMs = 100
	p, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	// Far-end tone
	far := make([]int16, 320)
	for i := range far {
		far[i] = int16(8000 * math.Sin(2*math.Pi*float64(i)/40))
	}
	// Near-end = delayed echo of far
	near := make([]int16, 320)
	delaySamples := cfg.EchoDelayMs * cfg.SampleRate / 1000
	// Warm up far ring
	for i := 0; i < 10; i++ {
		_ = p.ProcessRender(far)
	}
	for i := range near {
		near[i] = far[(i+len(far)-delaySamples%len(far))%len(far)] / 2
	}
	_, _ = p.ProcessCapture(near)
	// Process more frames
	var erle float64
	for i := 0; i < 20; i++ {
		_ = p.ProcessRender(far)
		out, err := p.ProcessCapture(near)
		if err != nil {
			t.Fatal(err)
		}
		erle = p.Stats().ERLE
		_ = out
	}
	if erle <= 0 {
		t.Fatalf("expected positive ERLE, got %v (backend=%s)", erle, p.Stats().Backend)
	}
}

func TestUnknownBackend(t *testing.T) {
	_, err := New(Config{Backend: "nope", SampleRate: 48000, FrameSamples: 960})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWebRTCUnavailable(t *testing.T) {
	_, err := New(Config{Backend: BackendWebRTC, SampleRate: 48000, FrameSamples: 960})
	if err == nil {
		t.Fatal("expected unavailable")
	}
}
