package audio

import (
	"encoding/hex"
	"strings"
)

// Backend names used in config and UI.
const (
	BackendAuto      = ""
	BackendALSA      = "alsa"
	BackendPulse     = "pulseaudio"
	BackendJACK      = "jack"
	BackendCoreAudio = "coreaudio"
	BackendDefault   = "default"
)

// Direction distinguishes capture and playback devices.
type Direction int

const (
	DirectionCapture Direction = iota
	DirectionPlayback
)

// Descriptor uniquely identifies a device across reboots when possible.
type Descriptor struct {
	Backend    string `json:"backend" yaml:"backend"`
	DeviceID   string `json:"device_id" yaml:"device_id"`     // hex of opaque ID
	DeviceName string `json:"device_name" yaml:"device_name"` // fallback
}

// DeviceInfo describes an enumerated audio device.
type DeviceInfo struct {
	Descriptor
	Name         string   `json:"name"`
	IsDefault    bool     `json:"is_default"`
	MaxChannels  int32    `json:"max_channels"`
	MinChannels  int32    `json:"min_channels"`
	SampleRates  []uint32 `json:"sample_rates,omitempty"`
	NativeFormat string   `json:"native_format,omitempty"`
}

// MatchRule reports how a descriptor was resolved.
type MatchRule string

const (
	MatchExactID MatchRule = "exact_id"
	MatchName    MatchRule = "name"
	MatchDefault MatchRule = "default"
	MatchNone    MatchRule = "none"
)

// Resolve picks a device from the list using exact ID, then name, then default.
func Resolve(want Descriptor, devices []DeviceInfo) (DeviceInfo, MatchRule) {
	if want.DeviceID != "" {
		for _, d := range devices {
			if sameBackend(want.Backend, d.Backend) && strings.EqualFold(d.DeviceID, want.DeviceID) {
				return d, MatchExactID
			}
		}
	}
	if want.DeviceName != "" {
		for _, d := range devices {
			if sameBackend(want.Backend, d.Backend) && d.DeviceName == want.DeviceName {
				return d, MatchName
			}
		}
		// Substring / case-insensitive name match as last name attempt.
		lower := strings.ToLower(want.DeviceName)
		for _, d := range devices {
			if sameBackend(want.Backend, d.Backend) && strings.Contains(strings.ToLower(d.Name), lower) {
				return d, MatchName
			}
		}
	}
	for _, d := range devices {
		if sameBackend(want.Backend, d.Backend) && d.IsDefault {
			return d, MatchDefault
		}
	}
	if len(devices) > 0 {
		// Prefer matching backend if any.
		for _, d := range devices {
			if sameBackend(want.Backend, d.Backend) {
				return d, MatchDefault
			}
		}
		return devices[0], MatchDefault
	}
	return DeviceInfo{}, MatchNone
}

func sameBackend(want, got string) bool {
	if want == "" || want == BackendAuto || want == BackendDefault {
		return true
	}
	return strings.EqualFold(want, got)
}

// EncodeDeviceID hex-encodes an opaque device ID blob.
func EncodeDeviceID(id []byte) string {
	if len(id) == 0 {
		return ""
	}
	return hex.EncodeToString(id)
}

// DecodeDeviceID decodes a hex device ID.
func DecodeDeviceID(s string) ([]byte, error) {
	if s == "" {
		return nil, nil
	}
	return hex.DecodeString(s)
}

// Enumerator lists devices. Implemented by the malgo-backed manager;
// tests use a fake.
type Enumerator interface {
	Backends() []string
	Devices(backend string, dir Direction) ([]DeviceInfo, error)
}
