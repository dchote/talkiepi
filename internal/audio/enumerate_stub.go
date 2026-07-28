//go:build !cgo

package audio

// NewMalgoEnumerator returns a placeholder when CGO is disabled.
func NewMalgoEnumerator() Enumerator {
	return StaticEnumerator{
		BackendList: append([]string{BackendAuto}, PlatformBackends()...),
		Capture: []DeviceInfo{{
			Descriptor:  Descriptor{Backend: BackendDefault, DeviceID: "default", DeviceName: "Default Input"},
			Name:        "Default Input",
			IsDefault:   true,
			MaxChannels: 1,
			MinChannels: 1,
		}},
		Playback: []DeviceInfo{{
			Descriptor:  Descriptor{Backend: BackendDefault, DeviceID: "default", DeviceName: "Default Output"},
			Name:        "Default Output",
			IsDefault:   true,
			MaxChannels: 2,
			MinChannels: 1,
		}},
	}
}
