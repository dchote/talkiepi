// Package led provides status indicator abstractions for talkiepi.
package led

// Indicator exposes online / participants / transmit status lights.
type Indicator interface {
	SetOnline(on bool)
	SetParticipants(on bool)
	SetTransmit(on bool)
	Close() error
}

// Memory is an in-memory Indicator for tests and headless runs.
type Memory struct {
	Online       bool
	Participants bool
	Transmit     bool
}

// NewMemory creates a Memory indicator.
func NewMemory() *Memory {
	return &Memory{}
}

func (m *Memory) SetOnline(on bool)       { m.Online = on }
func (m *Memory) SetParticipants(on bool) { m.Participants = on }
func (m *Memory) SetTransmit(on bool)     { m.Transmit = on }
func (m *Memory) Close() error            { return nil }

// GPIOConfig configures GPIO-backed LEDs.
type GPIOConfig struct {
	Enabled          bool
	Chip             string
	OnlineLine       int
	ParticipantsLine int
	TransmitLine     int
}

// GPIO is a GPIO-backed Indicator. When disabled or chip is empty it no-ops.
type GPIO struct {
	cfg    GPIOConfig
	online bool
	parts  bool
	tx     bool
}

// NewGPIO creates a GPIO indicator. When Enabled is false or Chip is empty,
// all setters are no-ops (still safe to use).
func NewGPIO(cfg GPIOConfig) *GPIO {
	return &GPIO{cfg: cfg}
}

func (g *GPIO) enabled() bool {
	return g.cfg.Enabled && g.cfg.Chip != ""
}

// SetOnline updates the online LED (no-op when disabled).
func (g *GPIO) SetOnline(on bool) {
	g.online = on
	if !g.enabled() {
		return
	}
	// Hardware drive is platform-specific; stub leaves state in memory only.
}

// SetParticipants updates the participants LED (no-op when disabled).
func (g *GPIO) SetParticipants(on bool) {
	g.parts = on
	if !g.enabled() {
		return
	}
}

// SetTransmit updates the transmit LED (no-op when disabled).
func (g *GPIO) SetTransmit(on bool) {
	g.tx = on
	if !g.enabled() {
		return
	}
}

// Close releases resources (no-op for stub).
func (g *GPIO) Close() error { return nil }

// State returns the last requested levels (useful in tests).
func (g *GPIO) State() (online, participants, transmit bool) {
	return g.online, g.parts, g.tx
}
