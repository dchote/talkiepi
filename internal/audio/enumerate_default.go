//go:build !cgo || !malgo

package audio

// NewMalgoEnumerator returns a StaticEnumerator placeholder when malgo is not linked.
// Build with -tags malgo (and CGO) for real device enumeration.
func NewMalgoEnumerator() Enumerator {
	return StaticEnumerator{
		BackendList: PlatformBackends(),
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
