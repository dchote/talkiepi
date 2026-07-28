package audio

import "testing"

func TestCommunicatorChimeEmbedded(t *testing.T) {
	pcm16 := CommunicatorChimePCM(16000)
	if len(pcm16) == 0 {
		t.Fatal("expected embedded communicator chime at 16 kHz")
	}
	// Source WAV is 6316 frames @ 16 kHz.
	if len(pcm16) < 6000 || len(pcm16) > 7000 {
		t.Fatalf("unexpected 16 kHz length %d", len(pcm16))
	}

	pcm48 := CommunicatorChimePCM(48000)
	if len(pcm48) == 0 {
		t.Fatal("expected resampled chime at 48 kHz")
	}
	// ~3x samples when going 16→48 kHz.
	ratio := float64(len(pcm48)) / float64(len(pcm16))
	if ratio < 2.8 || ratio > 3.2 {
		t.Fatalf("resample ratio=%v len16=%d len48=%d", ratio, len(pcm16), len(pcm48))
	}

	open := OpenChimePCM(48000)
	closePCM := CloseChimePCM(48000)
	if len(open) != len(closePCM) {
		t.Fatalf("open/close should be same asset: %d vs %d", len(open), len(closePCM))
	}
}

func TestParseWAVPCM16Mono(t *testing.T) {
	pcm, rate, err := parseWAVPCM16Mono(communicatorWAV)
	if err != nil {
		t.Fatal(err)
	}
	if rate != 16000 {
		t.Fatalf("rate=%d", rate)
	}
	if len(pcm) != 6316 {
		t.Fatalf("samples=%d want 6316", len(pcm))
	}
}
