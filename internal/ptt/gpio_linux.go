//go:build linux

package ptt

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/warthog618/go-gpiocdev"
)

type gpioSource struct {
	cfg    GPIOConfig
	mu     sync.Mutex
	line   *gpiocdev.Line
	out    chan<- Event
	ctx    context.Context
	cancel context.CancelFunc
}

// NewGPIO creates a GPIO PTT source using go-gpiocdev.
// If Chip is empty, returns an error.
func NewGPIO(cfg GPIOConfig) (Source, error) {
	if cfg.Chip == "" {
		return nil, errors.New("ptt: gpio chip path empty")
	}
	if cfg.Name == "" {
		cfg.Name = "gpio"
	}
	return &gpioSource{cfg: cfg}, nil
}

func (g *gpioSource) Name() string { return g.cfg.Name }

func (g *gpioSource) Start(ctx context.Context, out chan<- Event) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.line != nil {
		return errors.New("ptt: gpio already started")
	}
	if out == nil {
		return errors.New("ptt: nil out channel")
	}

	runCtx, cancel := context.WithCancel(ctx)
	g.ctx = runCtx
	g.cancel = cancel
	g.out = out

	opts := []gpiocdev.LineReqOption{
		gpiocdev.WithPullUp,
		gpiocdev.WithBothEdges,
		gpiocdev.WithEventHandler(g.onEdge),
	}
	if g.cfg.ActiveLow {
		opts = append(opts, gpiocdev.AsActiveLow)
	}

	line, err := gpiocdev.RequestLine(g.cfg.Chip, g.cfg.Line, opts...)
	if err != nil {
		cancel()
		g.ctx = nil
		g.cancel = nil
		g.out = nil
		return fmt.Errorf("ptt: request line: %w", err)
	}
	g.line = line

	// Emit initial level.
	if v, err := line.Value(); err == nil {
		g.emit(v != 0)
	}
	return nil
}

func (g *gpioSource) onEdge(evt gpiocdev.LineEvent) {
	pressed := evt.Type == gpiocdev.LineEventRisingEdge
	// With AsActiveLow, rising = became active.
	g.emit(pressed)
}

func (g *gpioSource) emit(pressed bool) {
	g.mu.Lock()
	out := g.out
	ctx := g.ctx
	name := g.cfg.Name
	g.mu.Unlock()
	if out == nil {
		return
	}
	ev := Event{Pressed: pressed, Source: name}
	if ctx == nil {
		select {
		case out <- ev:
		default:
		}
		return
	}
	select {
	case <-ctx.Done():
	case out <- ev:
	}
}

func (g *gpioSource) Stop() error {
	g.mu.Lock()
	cancel := g.cancel
	line := g.line
	g.cancel = nil
	g.line = nil
	g.out = nil
	g.ctx = nil
	g.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if line != nil {
		return line.Close()
	}
	return nil
}
