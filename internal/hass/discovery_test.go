package hass

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEntitiesParityWithEsp32(t *testing.T) {
	ents := Entities(Options{DeviceName: "Kitchen Pi", Version: "1.0.0"})
	need := map[string]string{
		"server": "text", "port": "text", "username": "text", "password": "text", "channel": "text",
		"mode": "select", "crypto": "select", "speaker_volume": "number",
		"mute": "switch", "ptt": "button", "communicator": "button",
		"connected": "binary_sensor", "voice_received": "binary_sensor", "voice_sending": "binary_sensor",
		"ping": "sensor", "transport": "sensor",
	}
	got := map[string]string{}
	for _, e := range ents {
		got[e.ObjectID] = e.Component
		if _, err := MarshalConfig(e); err != nil {
			t.Fatalf("marshal %s: %v", e.ObjectID, err)
		}
		topic := DiscoveryTopic("homeassistant", e.Component, "kitchen_pi", e.ObjectID)
		if !strings.Contains(topic, e.ObjectID) {
			t.Fatalf("topic=%s", topic)
		}
	}
	for id, comp := range need {
		if got[id] != comp {
			t.Fatalf("%s: got %q want %q", id, got[id], comp)
		}
	}
}

func TestSanitize(t *testing.T) {
	if sanitize("Kitchen Pi!") != "kitchen_pi" {
		t.Fatal(sanitize("Kitchen Pi!"))
	}
}

func TestDiscoveryPayloadHasDevice(t *testing.T) {
	ents := Entities(Options{DeviceName: "pi"})
	raw, _ := MarshalConfig(ents[0])
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["device"] == nil {
		t.Fatal("missing device")
	}
	if m["availability_topic"] == nil {
		t.Fatal("missing availability")
	}
}
