package audio

import "testing"

func TestMixerSumsAndClamps(t *testing.T) {
	m := NewMixer(2)
	m.Add([]int16{1000, 20000})
	m.Add([]int16{500, 20000})
	out := m.Render()
	if out[0] != 1500 {
		t.Fatalf("out[0]=%d", out[0])
	}
	if out[1] != 32767 {
		t.Fatalf("expected clamp, got %d", out[1])
	}
	// Cleared
	out = m.Render()
	if out[0] != 0 || out[1] != 0 {
		t.Fatalf("not cleared: %v", out)
	}
}
