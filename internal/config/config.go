package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/goccy/go-yaml"
)

// Config is the talkiepi runtime configuration.
type Config struct {
	Server   ServerConfig   `yaml:"server" json:"server"`
	Audio    AudioConfig    `yaml:"audio" json:"audio"`
	Voice    VoiceConfig    `yaml:"voice" json:"voice"`
	Triggers TriggersConfig `yaml:"triggers" json:"triggers"`
	LEDs     LEDsConfig     `yaml:"leds" json:"leds"`
	Web      WebConfig      `yaml:"web" json:"web"`
	MQTT     MQTTConfig     `yaml:"mqtt" json:"mqtt"`
	Hass     HassConfig     `yaml:"hass" json:"hass"`
	Updates  UpdatesConfig  `yaml:"updates" json:"updates"`
	Logging  LoggingConfig  `yaml:"logging" json:"logging"`
	mu       sync.RWMutex   `yaml:"-" json:"-"`
	path     string         `yaml:"-" json:"-"`
}

type ServerConfig struct {
	Address     string `yaml:"address" json:"address"`
	Port        int    `yaml:"port" json:"port"`
	Username    string `yaml:"username" json:"username"`
	Password    string `yaml:"password" json:"password"`
	Channel     string `yaml:"channel" json:"channel"`
	InsecureTLS bool   `yaml:"insecure_tls" json:"insecure_tls"`
	Certificate string `yaml:"certificate" json:"certificate"`
	Key         string `yaml:"key" json:"key"`
	Crypto      string `yaml:"crypto" json:"crypto"` // legacy | lite
}

type AudioConfig struct {
	Input      DeviceConfig     `yaml:"input" json:"input"`
	Output     DeviceConfig     `yaml:"output" json:"output"`
	Processing ProcessingConfig `yaml:"processing" json:"processing"`
}

type DeviceConfig struct {
	Backend      string  `yaml:"backend" json:"backend"`
	DeviceID     string  `yaml:"device_id" json:"device_id"`
	DeviceName   string  `yaml:"device_name" json:"device_name"`
	SampleRate   int     `yaml:"sample_rate" json:"sample_rate"`
	Channels     int     `yaml:"channels" json:"channels"`
	ChannelMap   []int   `yaml:"channel_map" json:"channel_map"`
	PeriodFrames int     `yaml:"period_frames" json:"period_frames"`
	Periods      int     `yaml:"periods" json:"periods"`
	Exclusive    bool    `yaml:"exclusive" json:"exclusive"`
	GainDB       float64 `yaml:"gain_db" json:"gain_db"`
	Volume       float64 `yaml:"volume" json:"volume"` // output only, 0-100
}

type ProcessingConfig struct {
	Backend       string  `yaml:"backend" json:"backend"` // speex | webrtc | none
	DuplexMode    string  `yaml:"duplex_mode" json:"duplex_mode"`
	EchoCancel    bool    `yaml:"echo_cancel" json:"echo_cancel"`
	EchoTailMs    int     `yaml:"echo_tail_ms" json:"echo_tail_ms"`
	EchoDelayMs   int     `yaml:"echo_delay_ms" json:"echo_delay_ms"`
	AdaptiveDelay bool    `yaml:"adaptive_delay" json:"adaptive_delay"`
	NoiseSuppress bool    `yaml:"noise_suppress" json:"noise_suppress"`
	NSLevel       string  `yaml:"ns_level" json:"ns_level"`
	AGC           bool    `yaml:"agc" json:"agc"`
	AGCTargetDBFS float64 `yaml:"agc_target_dbfs" json:"agc_target_dbfs"`
	Dereverb      bool    `yaml:"dereverb" json:"dereverb"`
	HighPass      bool    `yaml:"high_pass" json:"high_pass"`
}

type VoiceConfig struct {
	Mode       string `yaml:"mode" json:"mode"` // always_on | push_to_talk | communicator
	SampleRate int    `yaml:"sample_rate" json:"sample_rate"`
	Bitrate    int    `yaml:"bitrate" json:"bitrate"`
	FrameMs    int    `yaml:"frame_ms" json:"frame_ms"`
	Complexity int    `yaml:"complexity" json:"complexity"`
}

type TriggersConfig struct {
	GPIOChip    string `yaml:"gpio_chip" json:"gpio_chip"`
	GPIOLine    int    `yaml:"gpio_line" json:"gpio_line"`
	ActiveLow   bool   `yaml:"active_low" json:"active_low"`
	KeyboardKey string `yaml:"keyboard_key" json:"keyboard_key"`
	MQTTTopic   string `yaml:"mqtt_topic" json:"mqtt_topic"`
}

type LEDsConfig struct {
	Enabled          bool   `yaml:"enabled" json:"enabled"`
	Chip             string `yaml:"chip" json:"chip"`
	OnlineLine       int    `yaml:"online_line" json:"online_line"`
	ParticipantsLine int    `yaml:"participants_line" json:"participants_line"`
	TransmitLine     int    `yaml:"transmit_line" json:"transmit_line"`
}

type WebConfig struct {
	Host string `yaml:"host" json:"host"`
	Port int    `yaml:"port" json:"port"`
}

type MQTTConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	Broker   string `yaml:"broker" json:"broker"`
	Port     int    `yaml:"port" json:"port"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
	ClientID string `yaml:"client_id" json:"client_id"`
}

type HassConfig struct {
	Enabled         bool   `yaml:"enabled" json:"enabled"`
	DiscoveryPrefix string `yaml:"discovery_prefix" json:"discovery_prefix"`
	DeviceName      string `yaml:"device_name" json:"device_name"`
}

type UpdatesConfig struct {
	Enabled       bool   `yaml:"enabled" json:"enabled"`
	IntervalHours int    `yaml:"interval_hours" json:"interval_hours"`
	Repo          string `yaml:"repo" json:"repo"`
}

type LoggingConfig struct {
	Level string `yaml:"level" json:"level"`
}

// Defaults returns a Config with sane defaults.
func Defaults() *Config {
	return &Config{
		Server: ServerConfig{
			Address:     "",
			Port:        64738,
			Channel:     "Root",
			InsecureTLS: true,
			Crypto:      "legacy",
		},
		Audio: AudioConfig{
			Input: DeviceConfig{
				SampleRate:   48000,
				Channels:     1,
				ChannelMap:   []int{0},
				PeriodFrames: 960,
				Periods:      3,
			},
			Output: DeviceConfig{
				SampleRate:   48000,
				Channels:     1,
				ChannelMap:   []int{0},
				PeriodFrames: 960,
				Periods:      3,
				Volume:       65,
			},
			Processing: ProcessingConfig{
				Backend:       "speex",
				DuplexMode:    "auto",
				EchoCancel:    true,
				EchoTailMs:    150,
				AdaptiveDelay: true,
				NoiseSuppress: true,
				NSLevel:       "moderate",
				AGCTargetDBFS: -18,
				HighPass:      true,
			},
		},
		Voice: VoiceConfig{
			Mode:       "always_on",
			SampleRate: 48000,
			Bitrate:    40000,
			FrameMs:    20,
			Complexity: 5,
		},
		Triggers: TriggersConfig{
			ActiveLow: true,
		},
		LEDs: LEDsConfig{
			OnlineLine:       18,
			ParticipantsLine: 23,
			TransmitLine:     24,
		},
		Web: WebConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		MQTT: MQTTConfig{
			Port: 1883,
		},
		Hass: HassConfig{
			DiscoveryPrefix: "homeassistant",
			DeviceName:      "talkiepi",
		},
		Updates: UpdatesConfig{
			Enabled:       true,
			IntervalHours: 6,
			Repo:          "dchote/talkiepi",
		},
		Logging: LoggingConfig{Level: "info"},
	}
}

// Load reads YAML from path, applies env overrides, and validates.
func Load(path string) (*Config, error) {
	cfg := Defaults()
	cfg.path = path

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("config: read %s: %w", path, err)
			}
		} else if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("config: parse %s: %w", path, err)
		}
	}

	cfg.applyEnv()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Path returns the config file path.
func (c *Config) Path() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.path
}

// SetPath sets the config file path used by Save("").
func (c *Config) SetPath(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.path = path
}

// Clone returns a deep-ish copy safe for read-only use by callers.
func (c *Config) Clone() *Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := *c
	out.Audio.Input.ChannelMap = append([]int(nil), c.Audio.Input.ChannelMap...)
	out.Audio.Output.ChannelMap = append([]int(nil), c.Audio.Output.ChannelMap...)
	out.mu = sync.RWMutex{}
	return &out
}

// Save writes the config atomically to path (or c.path).
func (c *Config) Save(path string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if path == "" {
		path = c.path
	}
	if path == "" {
		return errors.New("config: no path")
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}
	return atomicWrite(path, data, 0o640)
}

// ApplyPatch merges a partial YAML document into the config and validates.
func (c *Config) ApplyPatch(patch []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	tmp := c.cloneUnlocked()
	if err := yaml.Unmarshal(patch, tmp); err != nil {
		return fmt.Errorf("config: patch: %w", err)
	}
	if err := tmp.Validate(); err != nil {
		return err
	}
	c.applyFrom(tmp)
	return nil
}

func (c *Config) cloneUnlocked() *Config {
	out := &Config{}
	out.applyFrom(c)
	return out
}

// applyFrom copies exported fields from src without touching c.mu.
func (c *Config) applyFrom(src *Config) {
	c.Server = src.Server
	c.Audio = src.Audio
	c.Audio.Input.ChannelMap = append([]int(nil), src.Audio.Input.ChannelMap...)
	c.Audio.Output.ChannelMap = append([]int(nil), src.Audio.Output.ChannelMap...)
	c.Voice = src.Voice
	c.Triggers = src.Triggers
	c.LEDs = src.LEDs
	c.Web = src.Web
	c.MQTT = src.MQTT
	c.Hass = src.Hass
	c.Updates = src.Updates
	c.Logging = src.Logging
	c.path = src.path
}

// Validate checks required fields and ranges.
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("config: server.port out of range: %d", c.Server.Port)
	}
	switch c.Server.Crypto {
	case "legacy", "lite", "":
	default:
		return fmt.Errorf("config: server.crypto invalid: %q", c.Server.Crypto)
	}
	switch c.Voice.Mode {
	case "always_on", "push_to_talk", "communicator", "":
	default:
		return fmt.Errorf("config: voice.mode invalid: %q", c.Voice.Mode)
	}
	switch c.Audio.Processing.Backend {
	case "speex", "webrtc", "none", "":
	default:
		return fmt.Errorf("config: audio.processing.backend invalid: %q", c.Audio.Processing.Backend)
	}
	if v := c.Audio.Output.Volume; v < 0 || v > 100 {
		return fmt.Errorf("config: audio.output.volume out of range: %v", v)
	}
	if c.Web.Port < 1 || c.Web.Port > 65535 {
		return fmt.Errorf("config: web.port out of range: %d", c.Web.Port)
	}
	if c.Audio.Processing.EchoTailMs < 0 || c.Audio.Processing.EchoTailMs > 500 {
		return fmt.Errorf("config: audio.processing.echo_tail_ms out of range: %d", c.Audio.Processing.EchoTailMs)
	}
	return nil
}

// ServerHostPort returns host:port for dialing.
func (c *Config) ServerHostPort() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return fmt.Sprintf("%s:%d", c.Server.Address, c.Server.Port)
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// applyEnv overlays TALKIEPI_* environment variables.
func (c *Config) applyEnv() {
	setStr := func(key string, dst *string) {
		if v, ok := os.LookupEnv(key); ok {
			*dst = v
		}
	}
	setInt := func(key string, dst *int) {
		if v, ok := os.LookupEnv(key); ok {
			if n, err := strconv.Atoi(v); err == nil {
				*dst = n
			}
		}
	}
	setBool := func(key string, dst *bool) {
		if v, ok := os.LookupEnv(key); ok {
			switch strings.ToLower(v) {
			case "1", "true", "yes", "on":
				*dst = true
			case "0", "false", "no", "off":
				*dst = false
			}
		}
	}
	setFloat := func(key string, dst *float64) {
		if v, ok := os.LookupEnv(key); ok {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				*dst = f
			}
		}
	}

	setStr("TALKIEPI_SERVER_ADDRESS", &c.Server.Address)
	setInt("TALKIEPI_SERVER_PORT", &c.Server.Port)
	setStr("TALKIEPI_SERVER_USERNAME", &c.Server.Username)
	setStr("TALKIEPI_SERVER_PASSWORD", &c.Server.Password)
	setStr("TALKIEPI_SERVER_CHANNEL", &c.Server.Channel)
	setBool("TALKIEPI_SERVER_INSECURE_TLS", &c.Server.InsecureTLS)
	setStr("TALKIEPI_SERVER_CRYPTO", &c.Server.Crypto)

	setStr("TALKIEPI_AUDIO_INPUT_BACKEND", &c.Audio.Input.Backend)
	setStr("TALKIEPI_AUDIO_INPUT_DEVICE_ID", &c.Audio.Input.DeviceID)
	setStr("TALKIEPI_AUDIO_INPUT_DEVICE_NAME", &c.Audio.Input.DeviceName)
	setInt("TALKIEPI_AUDIO_INPUT_SAMPLE_RATE", &c.Audio.Input.SampleRate)
	setFloat("TALKIEPI_AUDIO_INPUT_GAIN_DB", &c.Audio.Input.GainDB)

	setStr("TALKIEPI_AUDIO_OUTPUT_BACKEND", &c.Audio.Output.Backend)
	setStr("TALKIEPI_AUDIO_OUTPUT_DEVICE_ID", &c.Audio.Output.DeviceID)
	setStr("TALKIEPI_AUDIO_OUTPUT_DEVICE_NAME", &c.Audio.Output.DeviceName)
	setFloat("TALKIEPI_AUDIO_OUTPUT_VOLUME", &c.Audio.Output.Volume)

	setStr("TALKIEPI_AUDIO_PROCESSING_BACKEND", &c.Audio.Processing.Backend)
	setBool("TALKIEPI_AUDIO_PROCESSING_ECHO_CANCEL", &c.Audio.Processing.EchoCancel)
	setBool("TALKIEPI_AUDIO_PROCESSING_NOISE_SUPPRESS", &c.Audio.Processing.NoiseSuppress)
	setInt("TALKIEPI_AUDIO_PROCESSING_ECHO_TAIL_MS", &c.Audio.Processing.EchoTailMs)

	setStr("TALKIEPI_VOICE_MODE", &c.Voice.Mode)
	setInt("TALKIEPI_VOICE_SAMPLE_RATE", &c.Voice.SampleRate)
	setInt("TALKIEPI_VOICE_BITRATE", &c.Voice.Bitrate)

	setStr("TALKIEPI_TRIGGERS_GPIO_CHIP", &c.Triggers.GPIOChip)
	setInt("TALKIEPI_TRIGGERS_GPIO_LINE", &c.Triggers.GPIOLine)
	setBool("TALKIEPI_TRIGGERS_ACTIVE_LOW", &c.Triggers.ActiveLow)

	setStr("TALKIEPI_WEB_HOST", &c.Web.Host)
	setInt("TALKIEPI_WEB_PORT", &c.Web.Port)

	setBool("TALKIEPI_MQTT_ENABLED", &c.MQTT.Enabled)
	setStr("TALKIEPI_MQTT_BROKER", &c.MQTT.Broker)
	setInt("TALKIEPI_MQTT_PORT", &c.MQTT.Port)
	setStr("TALKIEPI_MQTT_USERNAME", &c.MQTT.Username)
	setStr("TALKIEPI_MQTT_PASSWORD", &c.MQTT.Password)

	setBool("TALKIEPI_HASS_ENABLED", &c.Hass.Enabled)
	setStr("TALKIEPI_HASS_DISCOVERY_PREFIX", &c.Hass.DiscoveryPrefix)
	setStr("TALKIEPI_HASS_DEVICE_NAME", &c.Hass.DeviceName)

	setBool("TALKIEPI_UPDATES_ENABLED", &c.Updates.Enabled)
	setInt("TALKIEPI_UPDATES_INTERVAL_HOURS", &c.Updates.IntervalHours)
	setStr("TALKIEPI_UPDATES_REPO", &c.Updates.Repo)

	setStr("TALKIEPI_LOGGING_LEVEL", &c.Logging.Level)
}
