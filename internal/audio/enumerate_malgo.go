//go:build cgo

package audio

import (
	"fmt"
	"sync"

	"github.com/gen2brain/malgo"
)

// MalgoEnumerator enumerates devices via miniaudio backends.
type MalgoEnumerator struct {
	mu sync.Mutex
}

// NewMalgoEnumerator returns a malgo-backed Enumerator.
func NewMalgoEnumerator() Enumerator {
	return &MalgoEnumerator{}
}

// Backends implements Enumerator. Leading empty string means auto/default.
func (e *MalgoEnumerator) Backends() []string {
	out := []string{BackendAuto}
	for _, b := range PlatformBackends() {
		if e.backendAvailable(b) {
			out = append(out, b)
		}
	}
	if len(out) == 1 {
		// Always expose the platform list even if probing failed.
		out = append(out, PlatformBackends()...)
	}
	return out
}

// Devices lists devices for a backend. Empty backend uses the default context.
func (e *MalgoEnumerator) Devices(backend string, dir Direction) ([]DeviceInfo, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	backends, err := malgoBackends(backend)
	if err != nil {
		return nil, err
	}
	ctx, err := malgo.InitContext(backends, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, fmt.Errorf("audio: init backend %q: %w", backendOrDefault(backend), err)
	}
	defer func() { _ = ctx.Uninit(); ctx.Free() }()

	dtype := malgo.Playback
	if dir == DirectionCapture {
		dtype = malgo.Capture
	}
	infos, err := ctx.Devices(dtype)
	if err != nil {
		return nil, fmt.Errorf("audio: list devices (%s): %w", backendOrDefault(backend), err)
	}

	resolvedBackend := backend
	if resolvedBackend == "" || resolvedBackend == BackendAuto {
		resolvedBackend = defaultBackendName()
	}

	out := make([]DeviceInfo, 0, len(infos))
	for _, info := range infos {
		full, ferr := ctx.DeviceInfo(dtype, info.ID, malgo.Shared)
		name := info.Name()
		src := info
		if ferr == nil {
			if n := full.Name(); n != "" {
				name = n
			}
			src = full
		}
		d := DeviceInfo{
			Descriptor: Descriptor{
				Backend:    resolvedBackend,
				DeviceID:   src.ID.String(),
				DeviceName: name,
			},
			Name:      name,
			IsDefault: src.IsDefault != 0,
		}
		d.MinChannels, d.MaxChannels, d.SampleRates, d.NativeFormat = formatCaps(src)
		out = append(out, d)
	}
	return out, nil
}

func (e *MalgoEnumerator) backendAvailable(backend string) bool {
	backends, err := malgoBackends(backend)
	if err != nil || len(backends) == 0 && backend != "" && backend != BackendAuto {
		return false
	}
	// Empty backends means "let miniaudio pick" — treat as available.
	if backend == "" || backend == BackendAuto {
		return true
	}
	ctx, err := malgo.InitContext(backends, malgo.ContextConfig{}, nil)
	if err != nil {
		return false
	}
	_ = ctx.Uninit()
	ctx.Free()
	return true
}

func malgoBackends(backend string) ([]malgo.Backend, error) {
	switch backend {
	case BackendAuto, BackendDefault: // BackendAuto is ""
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
	if b == "" || b == BackendAuto {
		return "auto"
	}
	return b
}

func defaultBackendName() string {
	bs := PlatformBackends()
	if len(bs) == 0 {
		return BackendDefault
	}
	return bs[0]
}

func formatCaps(info malgo.DeviceInfo) (minCh, maxCh int32, rates []uint32, native string) {
	minCh, maxCh = 1, 2
	seen := map[uint32]bool{}
	for i, f := range info.Formats {
		ch := int32(f.Channels)
		if i == 0 || ch < minCh {
			minCh = ch
		}
		if ch > maxCh {
			maxCh = ch
		}
		if f.SampleRate > 0 && !seen[f.SampleRate] {
			seen[f.SampleRate] = true
			rates = append(rates, f.SampleRate)
		}
		if native == "" {
			native = fmt.Sprintf("%v", f.Format)
		}
	}
	if minCh <= 0 {
		minCh = 1
	}
	if maxCh < minCh {
		maxCh = minCh
	}
	return minCh, maxCh, rates, native
}
