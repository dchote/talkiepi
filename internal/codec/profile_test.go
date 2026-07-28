package codec

import "testing"

func TestDefault(t *testing.T) {
	p := Default()
	if p.SampleRate != 48000 || p.Channels != 1 || p.FrameMs != 20 {
		t.Fatalf("unexpected default: %+v", p)
	}
	if p.Bitrate != 40000 || p.Complexity != 5 {
		t.Fatalf("unexpected default bitrate/complexity: %+v", p)
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	if p.FrameSamples() != 960 {
		t.Fatalf("FrameSamples = %d, want 960", p.FrameSamples())
	}
}

func TestEsp32Compatible(t *testing.T) {
	p := Esp32Compatible()
	if p.SampleRate != 16000 || p.Channels != 1 || p.FrameMs != 20 {
		t.Fatalf("unexpected esp32 profile: %+v", p)
	}
	if p.Bitrate != 16000 || p.Complexity != 1 {
		t.Fatalf("unexpected esp32 bitrate/complexity: %+v", p)
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	if p.FrameSamples() != 320 {
		t.Fatalf("FrameSamples = %d, want 320", p.FrameSamples())
	}
}

func TestValidateErrors(t *testing.T) {
	cases := []Profile{
		{SampleRate: 44100, Channels: 1, FrameMs: 20, Bitrate: 40000, Complexity: 5},
		{SampleRate: 48000, Channels: 3, FrameMs: 20, Bitrate: 40000, Complexity: 5},
		{SampleRate: 48000, Channels: 1, FrameMs: 15, Bitrate: 40000, Complexity: 5},
		{SampleRate: 48000, Channels: 1, FrameMs: 20, Bitrate: 100, Complexity: 5},
		{SampleRate: 48000, Channels: 1, FrameMs: 20, Bitrate: 40000, Complexity: 11},
	}
	for _, p := range cases {
		if err := p.Validate(); err == nil {
			t.Errorf("Validate(%+v) expected error", p)
		}
	}
}

func TestFrameSamplesZero(t *testing.T) {
	if (Profile{}).FrameSamples() != 0 {
		t.Fatal("expected 0")
	}
}
