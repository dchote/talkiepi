// Package ptt provides push-to-talk trigger sources and multiplexing.
package ptt

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Event is a press/release from a named source.
type Event struct {
	Pressed bool
	Source  string
}

// Source produces PTT events until stopped.
type Source interface {
	Name() string
	Start(ctx context.Context, out chan<- Event) error
	Stop() error
}

// Manual is a software PTT source for REST APIs and tests.
type Manual struct {
	name string
	mu   sync.Mutex
	out  chan<- Event
	ctx  context.Context
}

// NewManual creates a Manual source with the given name (default "manual").
func NewManual(name string) *Manual {
	if name == "" {
		name = "manual"
	}
	return &Manual{name: name}
}

func (m *Manual) Name() string { return m.name }

// Start arms the manual source to emit into out.
func (m *Manual) Start(ctx context.Context, out chan<- Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if out == nil {
		return errors.New("ptt: nil out channel")
	}
	m.out = out
	m.ctx = ctx
	return nil
}

// Stop disarms the source.
func (m *Manual) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.out = nil
	m.ctx = nil
	return nil
}

// Press emits a pressed event.
func (m *Manual) Press() error {
	return m.emit(true)
}

// Release emits a released event.
func (m *Manual) Release() error {
	return m.emit(false)
}

func (m *Manual) emit(pressed bool) error {
	m.mu.Lock()
	out := m.out
	ctx := m.ctx
	name := m.name
	m.mu.Unlock()
	if out == nil {
		return errors.New("ptt: manual source not started")
	}
	ev := Event{Pressed: pressed, Source: name}
	if ctx != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case out <- ev:
			return nil
		}
	}
	out <- ev
	return nil
}

// Mux merges events from multiple Sources into one channel.
type Mux struct {
	sources []Source
	mu      sync.Mutex
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
}

// NewMux creates a Mux over the given sources.
func NewMux(sources ...Source) *Mux {
	return &Mux{sources: append([]Source(nil), sources...)}
}

// Start starts all sources, forwarding their events to out.
func (m *Mux) Start(ctx context.Context, out chan<- Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started {
		return errors.New("ptt: mux already started")
	}
	if out == nil {
		return errors.New("ptt: nil out channel")
	}

	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	for _, src := range m.sources {
		src := src
		ch := make(chan Event, 8)
		if err := src.Start(ctx, ch); err != nil {
			cancel()
			for _, s := range m.sources {
				_ = s.Stop()
			}
			m.wg.Wait()
			m.cancel = nil
			return fmt.Errorf("ptt: start %s: %w", src.Name(), err)
		}
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case ev, ok := <-ch:
					if !ok {
						return
					}
					select {
					case <-ctx.Done():
						return
					case out <- ev:
					}
				}
			}
		}()
	}

	m.started = true
	return nil
}

// Stop stops all sources and waits for forwarders to exit.
func (m *Mux) Stop() error {
	m.mu.Lock()
	if !m.started {
		m.mu.Unlock()
		return nil
	}
	cancel := m.cancel
	sources := m.sources
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	var first error
	for _, s := range sources {
		if err := s.Stop(); err != nil && first == nil {
			first = err
		}
	}
	m.wg.Wait()

	m.mu.Lock()
	m.started = false
	m.cancel = nil
	m.mu.Unlock()
	return first
}

// Sources returns the configured sources.
func (m *Mux) Sources() []Source {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Source, len(m.sources))
	copy(out, m.sources)
	return out
}
