package audio

import "context"

// DeviceIO is a running capture and/or playback path that fills/drains PCM rings.
type DeviceIO interface {
	Start(ctx context.Context) error
	Stop() error
	Close() error
}

// DeviceOpenConfig configures capture/playback devices for a voice session.
type DeviceOpenConfig struct {
	Enumerator   Enumerator
	Input        Descriptor
	Output       Descriptor
	SampleRate   uint32
	Channels     uint32
	PeriodFrames uint32
	CaptureRing  *Ring
	PlaybackRing *Ring
}

// OpenDevices opens platform audio I/O (malgo when CGO is available).
func OpenDevices(cfg DeviceOpenConfig) (DeviceIO, error) {
	return openDevices(cfg)
}
