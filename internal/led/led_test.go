package led

import "testing"

func TestMemoryIndicator(t *testing.T) {
	var ind Indicator = NewMemory()
	ind.SetOnline(true)
	ind.SetParticipants(true)
	ind.SetTransmit(false)

	m := ind.(*Memory)
	if !m.Online || !m.Participants || m.Transmit {
		t.Fatalf("unexpected state: %+v", m)
	}
	m.SetTransmit(true)
	if !m.Transmit {
		t.Fatal("expected transmit on")
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestGPIODisabledNoOp(t *testing.T) {
	g := NewGPIO(GPIOConfig{Enabled: false, Chip: "gpiochip0"})
	g.SetOnline(true)
	g.SetParticipants(true)
	g.SetTransmit(true)
	on, parts, tx := g.State()
	if !on || !parts || !tx {
		t.Fatalf("state should still track requests: %v %v %v", on, parts, tx)
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestGPIOEmptyChipNoOp(t *testing.T) {
	g := NewGPIO(GPIOConfig{Enabled: true, Chip: ""})
	g.SetOnline(true)
	if !g.enabled() {
		// expected
	} else {
		t.Fatal("empty chip should disable hardware path")
	}
}
