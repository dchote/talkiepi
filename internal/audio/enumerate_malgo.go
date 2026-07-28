//go:build cgo && malgo

package audio

import (
	"fmt"

	"github.com/gen2brain/malgo"
)

// MalgoEnumerator enumerates devices via miniaudio backends.
type MalgoEnumerator struct{}

// NewMalgoEnumerator returns a malgo-backed Enumerator.
func NewMalgoEnumerator() *MalgoEnumerator { return &MalgoEnumerator{} }

// Backends implements Enumerator.
func (MalgoEnumerator) Backends() []string { return PlatformBackends() }

// Devices lists devices for a backend. Empty backend uses the default context.
func (MalgoEnumerator) Devices(backend string, dir Direction) ([]DeviceInfo, error) {
	backends, err := malgoBackends(backend)
	if err != nil {
		return nil, err
	}
	ctx, err := malgo.InitContext(backends, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, fmt.Errorf("audio: malgo init: %w", err)
	}
	defer func() { _ = ctx.Uninit(); ctx.Free() }()

	var dtype malgo.DeviceType
	if dir == DirectionCapture {
		dtype = malgo.Capture
	} else {
		dtype = malgo.Playback
	}
	infos, err := ctx.Devices(dtype)
	if err != nil {
		return nil, err
	}
	out := make([]DeviceInfo, 0, len(infos))
	for i, info := range infos {
		full, err := ctx.DeviceInfo(dtype, info.ID, malgo.Shared)
		name := info.Name()
		if err == nil && full.Name() != "" {
			name = full.Name()
		}
		d := DeviceInfo{
			Descriptor: Descriptor{
				Backend:    backendOrDefault(backend),
				DeviceID:   EncodeDeviceID(info.ID[:]),
				DeviceName: name,
			},
			Name:      name,
			IsDefault: i == 0,
		}
		if err == nil {
			d.MaxChannels = int32(full.MaxChannels)
			d.MinChannels = int32(full.MinChannels)
		}
		out = append(out, d)
	}
	return out, nil
}

func malgoBackends(backend string) ([]malgo.Backend, error) {
	switch backend {
	case "", BackendAuto, BackendDefault:
		return nil, nil
	case BackendALSA:
		return []malgo.Backend{malgo.BackendAlsa}, nil
	case BackendPulse:
		return []malgo.Backend{malgo.BackendPulseaudio}, nil
	case BackendJACK:
		return []malgo.Backend{malgo.BackendJack}, nil
	case BackendCoreAudio:
		return []malgo.Backend{malgo.BackendCoreaudio}, nil
	default:
		return nil, fmt.Errorf("audio: unknown backend %q", backend)
	}
}

func backendOrDefault(b string) string {
	if b == "" {
		return BackendDefault
	}
	return b
}
