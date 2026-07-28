//go:build !cgo

package audio

import (
	"context"
	"fmt"
)

type stubIO struct{}

func openDevices(DeviceOpenConfig) (DeviceIO, error) {
	return nil, fmt.Errorf("audio: device I/O requires CGO")
}

func (stubIO) Start(context.Context) error { return nil }
func (stubIO) Stop() error                 { return nil }
func (stubIO) Close() error                { return nil }
