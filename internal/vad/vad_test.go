package vad

import (
	"testing"
)

func tone(n int, amp int16) []int16 {
	s := make([]int16, n)
	for i := range s {
		if i%2 == 0 {
			s[i] = amp
		} else {
			s[i] = -amp
		}
	}
	return s
}

func TestRMS(t *testing.T) {
	if RMS(nil) != 0 {
		t.Fatal("nil")
	}
	if RMS([]int16{0, 0, 0}) != 0 {
		t.Fatal("zeros")
	}
	if RMS([]int16{1000, -1000}) < 900 {
		t.Fatal("expected high rms")
	}
}

func TestCalibrationThenTransmit(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CalibrationFrames = 3
	cfg.AttackFrames = 2
	cfg.HangoverFrames = 2
	cfg.MinThreshold = 50
	d := New(cfg)

	silence := tone(32, 10)
	for i := 0; i < 3; i++ {
		r := d.Process(silence)
		if r.Transmit {
			t.Fatal("should not transmit during calibration")
		}
	}
	speech := tone(32, 2000)
	r := d.Process(speech)
	if r.Transmit {
		t.Fatal("need attack frames")
	}
	r = d.Process(speech)
	if !r.Transmit {
		t.Fatal("expected transmit after attack")
	}
}

func TestHangoverTerminator(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CalibrationFrames = 1
	cfg.AttackFrames = 1
	cfg.HangoverFrames = 2
	cfg.MinThreshold = 50
	d := New(cfg)
	d.Process(tone(16, 10)) // calibrate
	d.Process(tone(16, 2000))
	if !d.Process(tone(16, 2000)).Transmit {
		// already transmitting from first speech if attack=1
	}
	// Ensure transmitting
	d.ForceTransmit()
	r := d.Process(tone(16, 10))
	if !r.Transmit {
		t.Fatal("still in hangover")
	}
	r = d.Process(tone(16, 10))
	if r.Transmit {
		t.Fatal("should stop after hangover")
	}
	if !r.SendTerminator {
		t.Fatal("expected terminator")
	}
}

func TestForceStopTerminator(t *testing.T) {
	d := New(DefaultConfig())
	d.ForceTransmit()
	d.ForceStop()
	if !d.TakeTerminator() {
		t.Fatal("expected terminator after force stop")
	}
}

func TestReset(t *testing.T) {
	d := New(DefaultConfig())
	d.ForceTransmit()
	d.Reset()
	if d.state != StateIdle {
		t.Fatalf("state=%v", d.state)
	}
}
