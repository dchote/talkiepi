package communicator

import (
	"testing"
	"time"
)

func TestIdleToggleOpenChime(t *testing.T) {
	c := New()
	if c.State() != Idle {
		t.Fatalf("initial state = %v, want Idle", c.State())
	}
	c.Toggle()
	if c.State() != OpenChime {
		t.Fatalf("after Toggle = %v, want OpenChime", c.State())
	}
	if !c.ShouldSuppressRX() {
		t.Fatal("ShouldSuppressRX should be true during OpenChime")
	}
	if c.ShouldTransmit() {
		t.Fatal("ShouldTransmit should be false during OpenChime")
	}
}

func TestOpenChimeDoneToMicActive(t *testing.T) {
	c := New()
	c.Toggle()
	c.OnChimeDone()
	if c.State() != MicActive {
		t.Fatalf("state = %v, want MicActive", c.State())
	}
	if !c.ShouldTransmit() {
		t.Fatal("ShouldTransmit should be true in MicActive")
	}
	if !c.ShouldSuppressRX() {
		t.Fatal("ShouldSuppressRX should be true in MicActive")
	}
}

func TestMicActiveSilenceStartsWindow(t *testing.T) {
	c := New()
	c.Toggle()
	c.OnChimeDone()
	c.OnVoiceActivity(true)
	if c.State() != MicActive {
		t.Fatalf("speech should stay MicActive, got %v", c.State())
	}
	c.OnVoiceActivity(false)
	if c.State() != SilenceWindow {
		t.Fatalf("state = %v, want SilenceWindow", c.State())
	}
	if !c.ShouldTransmit() {
		t.Fatal("ShouldTransmit should remain true in SilenceWindow")
	}
}

func TestSilenceWindowSpeechReturnsMicActive(t *testing.T) {
	c := New()
	c.Toggle()
	c.OnChimeDone()
	c.OnVoiceActivity(false)
	c.OnVoiceActivity(true)
	if c.State() != MicActive {
		t.Fatalf("state = %v, want MicActive", c.State())
	}
}

func TestSilenceWindowTimeoutCloses(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := New(WithNow(func() time.Time { return now }))
	c.Toggle()
	c.OnChimeDone()
	c.OnVoiceActivity(false) // enter SilenceWindow at t=0
	if c.State() != SilenceWindow {
		t.Fatalf("state = %v, want SilenceWindow", c.State())
	}

	now = now.Add(SilenceTimeout - time.Millisecond)
	c.OnVoiceActivity(false)
	if c.State() != SilenceWindow {
		t.Fatalf("before timeout state = %v, want SilenceWindow", c.State())
	}

	now = now.Add(2 * time.Millisecond)
	c.OnVoiceActivity(false)
	if c.State() != CloseChime {
		t.Fatalf("after timeout state = %v, want CloseChime", c.State())
	}
	if c.ShouldTransmit() {
		t.Fatal("ShouldTransmit should be false in CloseChime")
	}
}

func TestCloseChimeDoneToIdle(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := New(WithNow(func() time.Time { return now }))
	c.Toggle()
	c.OnChimeDone()
	c.OnVoiceActivity(false)
	now = now.Add(SilenceTimeout)
	c.OnVoiceActivity(false)
	if c.State() != CloseChime {
		t.Fatalf("state = %v, want CloseChime", c.State())
	}
	c.OnChimeDone()
	if c.State() != Idle {
		t.Fatalf("state = %v, want Idle", c.State())
	}
	if c.ShouldSuppressRX() {
		t.Fatal("ShouldSuppressRX should be false in Idle")
	}
	if c.ShouldTransmit() {
		t.Fatal("ShouldTransmit should be false in Idle")
	}
}

func TestCancelFromAnyState(t *testing.T) {
	t.Run("Idle", func(t *testing.T) {
		c := New()
		c.Cancel()
		assertState(t, c, Idle)
	})
	t.Run("OpenChime", func(t *testing.T) {
		c := New()
		c.Toggle()
		c.Cancel()
		assertState(t, c, Idle)
	})
	t.Run("MicActive", func(t *testing.T) {
		c := New()
		c.Toggle()
		c.OnChimeDone()
		c.Cancel()
		assertState(t, c, Idle)
	})
	t.Run("SilenceWindow", func(t *testing.T) {
		c := New()
		c.Toggle()
		c.OnChimeDone()
		c.OnVoiceActivity(false)
		c.Cancel()
		assertState(t, c, Idle)
	})
	t.Run("CloseChime", func(t *testing.T) {
		now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		c := New(WithNow(func() time.Time { return now }))
		c.Toggle()
		c.OnChimeDone()
		c.OnVoiceActivity(false)
		now = now.Add(SilenceTimeout)
		c.OnVoiceActivity(false)
		assertState(t, c, CloseChime)
		c.Cancel()
		assertState(t, c, Idle)
	})
}

func TestToggleFromOpenGoesToCloseChime(t *testing.T) {
	c := New()
	c.Toggle()
	c.OnChimeDone()
	c.Toggle()
	if c.State() != CloseChime {
		t.Fatalf("state = %v, want CloseChime", c.State())
	}
}

func TestOnChimeDoneIgnoredInOtherStates(t *testing.T) {
	c := New()
	c.OnChimeDone()
	if c.State() != Idle {
		t.Fatalf("state = %v, want Idle", c.State())
	}
	c.Toggle()
	c.OnChimeDone() // -> MicActive
	c.OnChimeDone() // no-op
	if c.State() != MicActive {
		t.Fatalf("state = %v, want MicActive", c.State())
	}
}

func TestFullCycle(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	c := New(WithNow(func() time.Time { return now }))

	c.Toggle()
	assertState(t, c, OpenChime)
	c.OnChimeDone()
	assertState(t, c, MicActive)
	c.OnVoiceActivity(true)
	assertState(t, c, MicActive)
	c.OnVoiceActivity(false)
	assertState(t, c, SilenceWindow)
	c.OnVoiceActivity(true)
	assertState(t, c, MicActive)
	c.OnVoiceActivity(false)
	assertState(t, c, SilenceWindow)
	now = now.Add(SilenceTimeout)
	c.OnVoiceActivity(false)
	assertState(t, c, CloseChime)
	c.OnChimeDone()
	assertState(t, c, Idle)
}

func TestStateString(t *testing.T) {
	cases := []struct {
		s    State
		want string
	}{
		{Idle, "Idle"},
		{OpenChime, "OpenChime"},
		{MicActive, "MicActive"},
		{SilenceWindow, "SilenceWindow"},
		{CloseChime, "CloseChime"},
		{State(99), "Unknown"},
	}
	for _, tc := range cases {
		if got := tc.s.String(); got != tc.want {
			t.Errorf("%v.String() = %q, want %q", tc.s, got, tc.want)
		}
	}
}

func assertState(t *testing.T, c *Communicator, want State) {
	t.Helper()
	if got := c.State(); got != want {
		t.Fatalf("state = %v, want %v", got, want)
	}
}
