// Package communicator implements half-duplex Communicator mode state machine
// (matching esp32-mumble behavior).
package communicator

import (
	"sync"
	"time"
)

// SilenceTimeout is how long silence must persist before closing the channel.
const SilenceTimeout = 2 * time.Second

// State is a Communicator mode state.
type State int

const (
	Idle State = iota
	OpenChime
	MicActive
	SilenceWindow
	CloseChime
)

func (s State) String() string {
	switch s {
	case Idle:
		return "Idle"
	case OpenChime:
		return "Open Channel"
	case MicActive:
		return "MicActive"
	case SilenceWindow:
		return "SilenceWindow"
	case CloseChime:
		return "Close Channel"
	default:
		return "Unknown"
	}
}

// Communicator is a half-duplex voice channel state machine.
type Communicator struct {
	mu             sync.Mutex
	state          State
	silenceStarted time.Time
	now            func() time.Time
}

// Option configures a Communicator.
type Option func(*Communicator)

// WithNow injects a clock for tests.
func WithNow(now func() time.Time) Option {
	return func(c *Communicator) { c.now = now }
}

// New creates a Communicator in Idle.
func New(opts ...Option) *Communicator {
	c := &Communicator{
		state: Idle,
		now:   time.Now,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// State returns the current state.
func (c *Communicator) State() State {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// ShouldTransmit reports whether the microphone path should send audio.
func (c *Communicator) ShouldTransmit() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state == MicActive || c.state == SilenceWindow
}

// ShouldSuppressRX reports whether receive audio should be muted (half-duplex).
func (c *Communicator) ShouldSuppressRX() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state != Idle
}

// Toggle starts a session from Idle, or begins close from an open state.
func (c *Communicator) Toggle() {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch c.state {
	case Idle:
		c.state = OpenChime
	case OpenChime, MicActive, SilenceWindow:
		c.state = CloseChime
		c.silenceStarted = time.Time{}
	}
}

// Cancel forces Idle from any state.
func (c *Communicator) Cancel() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = Idle
	c.silenceStarted = time.Time{}
}

// OnChimeDone advances past open/close chime playback.
func (c *Communicator) OnChimeDone() {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch c.state {
	case OpenChime:
		c.state = MicActive
	case CloseChime:
		c.state = Idle
		c.silenceStarted = time.Time{}
	}
}

// OnVoiceActivity updates the machine based on speech detection.
// Call periodically while the session is open so silence timeout can fire.
func (c *Communicator) OnVoiceActivity(speech bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()

	switch c.state {
	case MicActive:
		if !speech {
			c.state = SilenceWindow
			c.silenceStarted = now
		}
	case SilenceWindow:
		if speech {
			c.state = MicActive
			c.silenceStarted = time.Time{}
			return
		}
		if !c.silenceStarted.IsZero() && now.Sub(c.silenceStarted) >= SilenceTimeout {
			c.state = CloseChime
			c.silenceStarted = time.Time{}
		}
	}
}
