//go:build !linux

package ptt

import (
	"context"
	"errors"
)

// gpioSource is a no-op PTT source used on non-Linux platforms.
type gpioSource struct {
	name string
}

// NewGPIO returns a no-op GPIO source on non-Linux builds.
// If Chip is empty, returns an error.
func NewGPIO(cfg GPIOConfig) (Source, error) {
	if cfg.Chip == "" {
		return nil, errors.New("ptt: gpio chip path empty")
	}
	name := cfg.Name
	if name == "" {
		name = "gpio"
	}
	return &gpioSource{name: name}, nil
}

func (g *gpioSource) Name() string { return g.name }

func (g *gpioSource) Start(ctx context.Context, out chan<- Event) error {
	return nil
}

func (g *gpioSource) Stop() error { return nil }
