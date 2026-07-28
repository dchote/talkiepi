package audio

import (
	"fmt"
	"runtime"
)

// PlatformBackends returns backends typically available on this OS.
// The empty string (BackendAuto) is not included; callers that want an
// "auto" choice should prepend it.
func PlatformBackends() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{BackendCoreAudio}
	case "linux":
		return []string{BackendPulse, BackendALSA, BackendJACK}
	default:
		return []string{BackendDefault}
	}
}

// BackendLabels maps backend IDs to UI labels.
func BackendLabels() map[string]string {
	return map[string]string{
		BackendAuto:      "Auto (system default)",
		BackendDefault:   "Default",
		BackendALSA:      "ALSA",
		BackendPulse:     "PulseAudio",
		BackendJACK:      "JACK",
		BackendCoreAudio: "CoreAudio",
	}
}

// StaticEnumerator is a test/dev enumerator with injected devices.
type StaticEnumerator struct {
	BackendList []string
	Capture     []DeviceInfo
	Playback    []DeviceInfo
}

// Backends implements Enumerator.
func (s StaticEnumerator) Backends() []string {
	if len(s.BackendList) == 0 {
		return PlatformBackends()
	}
	return s.BackendList
}

// Devices implements Enumerator.
func (s StaticEnumerator) Devices(_ string, dir Direction) ([]DeviceInfo, error) {
	switch dir {
	case DirectionCapture:
		return append([]DeviceInfo(nil), s.Capture...), nil
	case DirectionPlayback:
		return append([]DeviceInfo(nil), s.Playback...), nil
	default:
		return nil, fmt.Errorf("audio: unknown direction %d", dir)
	}
}
