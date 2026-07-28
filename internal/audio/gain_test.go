package audio

import "testing"

func TestGainLinearFromDB(t *testing.T) {
	if g := GainLinearFromDB(0); g != 1 {
		t.Fatalf("0 dB => %v", g)
	}
	if g := GainLinearFromDB(6); g < 1.9 || g > 2.1 {
		t.Fatalf("6 dB => %v", g)
	}
}

func TestVolumeLinearFromPercent(t *testing.T) {
	if v := VolumeLinearFromPercent(100); v != 1 {
		t.Fatalf("100 => %v", v)
	}
	if v := VolumeLinearFromPercent(50); v != 0.5 {
		t.Fatalf("50 => %v", v)
	}
	if v := VolumeLinearFromPercent(-1); v != 0 {
		t.Fatalf("clamp low => %v", v)
	}
}

func TestApplyLinearGain(t *testing.T) {
	pcm := []int16{1000, -1000}
	ApplyLinearGain(pcm, 2)
	if pcm[0] != 2000 || pcm[1] != -2000 {
		t.Fatalf("%v", pcm)
	}
	ApplyLinearGain(pcm, 0)
	if pcm[0] != 0 || pcm[1] != 0 {
		t.Fatalf("zero: %v", pcm)
	}
}
