package hass

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Device describes the Home Assistant device block.
type Device struct {
	Identifiers  []string `json:"identifiers"`
	Name         string   `json:"name"`
	Manufacturer string   `json:"manufacturer"`
	Model        string   `json:"model"`
	SWVersion    string   `json:"sw_version,omitempty"`
}

// Entity is a single MQTT discovery config payload.
type Entity struct {
	Component string
	ObjectID  string
	Config    map[string]any
}

// Options configure discovery topic generation.
type Options struct {
	Prefix     string
	DeviceName string
	NodeID     string
	Version    string
	BaseTopic  string // state/command root, e.g. talkiepi/<node>
}

// Entities returns discovery configs mirroring esp32-mumble's entity set.
func Entities(opt Options) []Entity {
	if opt.Prefix == "" {
		opt.Prefix = "homeassistant"
	}
	if opt.DeviceName == "" {
		opt.DeviceName = "talkiepi"
	}
	if opt.NodeID == "" {
		opt.NodeID = sanitize(opt.DeviceName)
	}
	if opt.BaseTopic == "" {
		opt.BaseTopic = "talkiepi/" + opt.NodeID
	}
	dev := Device{
		Identifiers:  []string{"talkiepi_" + opt.NodeID},
		Name:         opt.DeviceName,
		Manufacturer: "dchote",
		Model:        "talkiepi",
		SWVersion:    opt.Version,
	}
	avail := opt.BaseTopic + "/status"
	base := func(extra map[string]any) map[string]any {
		m := map[string]any{
			"availability_topic":    avail,
			"payload_available":     "online",
			"payload_not_available": "offline",
			"device":                dev,
			"unique_id":             fmt.Sprintf("talkiepi_%s_%s", opt.NodeID, extra["object_id"]),
		}
		for k, v := range extra {
			if k == "object_id" {
				continue
			}
			m[k] = v
		}
		return m
	}

	var out []Entity
	add := func(component, objectID string, cfg map[string]any) {
		cfg["object_id"] = objectID
		cfg["name"] = cfg["name"]
		out = append(out, Entity{Component: component, ObjectID: objectID, Config: base(cfg)})
	}

	// Config texts (esp32-mumble numbered names)
	for _, e := range []struct{ id, name, topic string }{
		{"server", "1. Server", "server"},
		{"port", "2. Port", "port"},
		{"username", "3. Username", "username"},
		{"password", "4. Password", "password"},
		{"channel", "5. Channel", "channel"},
	} {
		cfg := map[string]any{
			"object_id":       e.id,
			"name":            e.name,
			"command_topic":   opt.BaseTopic + "/config/" + e.topic + "/set",
			"state_topic":     opt.BaseTopic + "/config/" + e.topic,
			"entity_category": "config",
		}
		if e.id == "password" {
			cfg["command_template"] = "{{ value }}"
		}
		add("text", e.id, cfg)
	}

	add("select", "mode", map[string]any{
		"object_id":       "mode",
		"name":            "6. Mode",
		"command_topic":   opt.BaseTopic + "/config/mode/set",
		"state_topic":     opt.BaseTopic + "/config/mode",
		"options":         []string{"always_on", "push_to_talk", "communicator"},
		"entity_category": "config",
	})
	add("select", "crypto", map[string]any{
		"object_id":       "crypto",
		"name":            "7. Crypto",
		"command_topic":   opt.BaseTopic + "/config/crypto/set",
		"state_topic":     opt.BaseTopic + "/config/crypto",
		"options":         []string{"legacy", "lite"},
		"entity_category": "config",
	})
	add("number", "speaker_volume", map[string]any{
		"object_id":       "speaker_volume",
		"name":            "Speaker Volume",
		"command_topic":   opt.BaseTopic + "/config/speaker_volume/set",
		"state_topic":     opt.BaseTopic + "/config/speaker_volume",
		"min":             0,
		"max":             100,
		"step":            5,
		"entity_category": "config",
	})

	add("switch", "mute", map[string]any{
		"object_id":     "mute",
		"name":          "Mute",
		"command_topic": opt.BaseTopic + "/mute/set",
		"state_topic":   opt.BaseTopic + "/mute",
		"payload_on":    "ON",
		"payload_off":   "OFF",
	})
	add("button", "ptt", map[string]any{
		"object_id":     "ptt",
		"name":          "PTT",
		"command_topic": opt.BaseTopic + "/ptt/set",
		"payload_press": "PRESS",
	})
	add("button", "communicator", map[string]any{
		"object_id":     "communicator",
		"name":          "Communicator Toggle",
		"command_topic": opt.BaseTopic + "/communicator/set",
		"payload_press": "PRESS",
	})
	add("button", "reconnect", map[string]any{
		"object_id":       "reconnect",
		"name":            "Reconnect",
		"command_topic":   opt.BaseTopic + "/reconnect/set",
		"payload_press":   "PRESS",
		"entity_category": "diagnostic",
	})
	add("button", "reset_config", map[string]any{
		"object_id":       "reset_config",
		"name":            "Reset Config",
		"command_topic":   opt.BaseTopic + "/reset_config/set",
		"payload_press":   "PRESS",
		"entity_category": "diagnostic",
	})

	add("binary_sensor", "connected", map[string]any{
		"object_id":       "connected",
		"name":            "Mumble Connected",
		"state_topic":     opt.BaseTopic + "/connected",
		"payload_on":      "ON",
		"payload_off":     "OFF",
		"device_class":    "connectivity",
		"entity_category": "diagnostic",
	})
	add("binary_sensor", "voice_received", map[string]any{
		"object_id":    "voice_received",
		"name":         "Voice Received",
		"state_topic":  opt.BaseTopic + "/voice_received",
		"payload_on":   "ON",
		"payload_off":  "OFF",
		"device_class": "running",
	})
	add("binary_sensor", "voice_sending", map[string]any{
		"object_id":    "voice_sending",
		"name":         "Voice Sending",
		"state_topic":  opt.BaseTopic + "/voice_sending",
		"payload_on":   "ON",
		"payload_off":  "OFF",
		"device_class": "running",
	})
	add("sensor", "ping", map[string]any{
		"object_id":           "ping",
		"name":                "Mumble Ping",
		"state_topic":         opt.BaseTopic + "/ping",
		"unit_of_measurement": "ms",
		"device_class":        "duration",
		"entity_category":     "diagnostic",
	})
	add("sensor", "transport", map[string]any{
		"object_id":       "transport",
		"name":            "Transport",
		"state_topic":     opt.BaseTopic + "/transport",
		"entity_category": "diagnostic",
	})
	add("sensor", "channel_name", map[string]any{
		"object_id":       "channel_name",
		"name":            "Current Channel",
		"state_topic":     opt.BaseTopic + "/channel_name",
		"entity_category": "diagnostic",
	})
	add("sensor", "participants", map[string]any{
		"object_id":       "participants",
		"name":            "Participants",
		"state_topic":     opt.BaseTopic + "/participants",
		"entity_category": "diagnostic",
	})
	add("sensor", "erle", map[string]any{
		"object_id":           "erle",
		"name":                "Echo Return Loss",
		"state_topic":         opt.BaseTopic + "/erle",
		"unit_of_measurement": "dB",
		"entity_category":     "diagnostic",
	})

	return out
}

// DiscoveryTopic returns the MQTT discovery config topic for an entity.
func DiscoveryTopic(prefix, component, nodeID, objectID string) string {
	return fmt.Sprintf("%s/%s/%s/%s/config", prefix, component, nodeID, objectID)
}

// MarshalConfig JSON-encodes an entity config.
func MarshalConfig(e Entity) ([]byte, error) {
	return json.Marshal(e.Config)
}

func sanitize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		} else if r == ' ' {
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "talkiepi"
	}
	return b.String()
}
