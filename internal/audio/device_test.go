package audio

import (
	"testing"
)

func TestResolveExactID(t *testing.T) {
	devs := []DeviceInfo{
		{Descriptor: Descriptor{Backend: "alsa", DeviceID: "aa", DeviceName: "A"}, Name: "A"},
		{Descriptor: Descriptor{Backend: "alsa", DeviceID: "bb", DeviceName: "B"}, Name: "B", IsDefault: true},
	}
	got, rule := Resolve(Descriptor{Backend: "alsa", DeviceID: "bb"}, devs)
	if rule != MatchExactID || got.DeviceID != "bb" {
		t.Fatalf("got %+v rule=%s", got, rule)
	}
}

func TestResolveNameFallback(t *testing.T) {
	devs := []DeviceInfo{
		{Descriptor: Descriptor{Backend: "pulseaudio", DeviceID: "1", DeviceName: "USB Speakerphone"}, Name: "USB Speakerphone"},
	}
	got, rule := Resolve(Descriptor{DeviceName: "USB Speakerphone"}, devs)
	if rule != MatchName || got.DeviceName != "USB Speakerphone" {
		t.Fatalf("got %+v rule=%s", got, rule)
	}
}

func TestResolveDefault(t *testing.T) {
	devs := []DeviceInfo{
		{Descriptor: Descriptor{Backend: "alsa", DeviceID: "1"}, Name: "hw:0", IsDefault: false},
		{Descriptor: Descriptor{Backend: "alsa", DeviceID: "2"}, Name: "hw:1", IsDefault: true},
	}
	got, rule := Resolve(Descriptor{}, devs)
	if rule != MatchDefault || !got.IsDefault {
		t.Fatalf("got %+v rule=%s", got, rule)
	}
}

func TestResolveNone(t *testing.T) {
	_, rule := Resolve(Descriptor{DeviceID: "x"}, nil)
	if rule != MatchNone {
		t.Fatalf("rule=%s", rule)
	}
}

func TestResolveBackendFilter(t *testing.T) {
	devs := []DeviceInfo{
		{Descriptor: Descriptor{Backend: "alsa", DeviceID: "1"}, Name: "a", IsDefault: true},
		{Descriptor: Descriptor{Backend: "pulseaudio", DeviceID: "2"}, Name: "p", IsDefault: true},
	}
	got, _ := Resolve(Descriptor{Backend: "pulseaudio"}, devs)
	if got.Backend != "pulseaudio" {
		t.Fatalf("got %s", got.Backend)
	}
}

func TestEncodeDecodeDeviceID(t *testing.T) {
	id := []byte{0xde, 0xad, 0xbe, 0xef}
	s := EncodeDeviceID(id)
	got, err := DecodeDeviceID(s)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(id) {
		t.Fatalf("got %v", got)
	}
}
