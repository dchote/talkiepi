package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultsValid(t *testing.T) {
	cfg := Defaults()
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadMissingUsesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.yaml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != 64738 {
		t.Fatalf("port=%d", cfg.Server.Port)
	}
	if cfg.Voice.Mode != "always_on" {
		t.Fatalf("mode=%s", cfg.Voice.Mode)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "talkiepi.yaml")
	cfg := Defaults()
	cfg.Server.Address = "mumble.example.com"
	cfg.Server.Username = "pi-1"
	cfg.Voice.Mode = "communicator"
	cfg.Audio.Output.Volume = 80
	cfg.path = path
	if err := cfg.Save(""); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Server.Address != "mumble.example.com" || got.Server.Username != "pi-1" {
		t.Fatalf("server: %+v", got.Server)
	}
	if got.Voice.Mode != "communicator" || got.Audio.Output.Volume != 80 {
		t.Fatalf("voice/audio: mode=%s vol=%v", got.Voice.Mode, got.Audio.Output.Volume)
	}
}

func TestApplyPatch(t *testing.T) {
	cfg := Defaults()
	if err := cfg.ApplyPatch([]byte("server:\n  address: 10.0.0.5\n  port: 64739\n")); err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Address != "10.0.0.5" || cfg.Server.Port != 64739 {
		t.Fatalf("got %+v", cfg.Server)
	}
	// Unpatched fields preserved
	if cfg.Server.Channel != "Root" {
		t.Fatalf("channel=%s", cfg.Server.Channel)
	}
}

func TestApplyPatchInvalid(t *testing.T) {
	cfg := Defaults()
	if err := cfg.ApplyPatch([]byte("voice:\n  mode: nonsense\n")); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestEnvOverrides(t *testing.T) {
	t.Setenv("TALKIEPI_SERVER_ADDRESS", "env.example")
	t.Setenv("TALKIEPI_SERVER_PORT", "12345")
	t.Setenv("TALKIEPI_VOICE_MODE", "push_to_talk")
	t.Setenv("TALKIEPI_AUDIO_PROCESSING_ECHO_CANCEL", "false")
	t.Setenv("TALKIEPI_WEB_PORT", "9090")

	cfg, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Address != "env.example" || cfg.Server.Port != 12345 {
		t.Fatalf("server: %+v", cfg.Server)
	}
	if cfg.Voice.Mode != "push_to_talk" {
		t.Fatalf("mode=%s", cfg.Voice.Mode)
	}
	if cfg.Audio.Processing.EchoCancel {
		t.Fatal("echo_cancel should be false")
	}
	if cfg.Web.Port != 9090 {
		t.Fatalf("web.port=%d", cfg.Web.Port)
	}
}

func TestValidateRejectsBadPort(t *testing.T) {
	cfg := Defaults()
	cfg.Server.Port = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestAtomicSaveDoesNotLeaveTmp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	cfg := Defaults()
	cfg.path = path
	if err := cfg.Save(""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("tmp file left behind: %v", err)
	}
}

func TestCloneIndependence(t *testing.T) {
	cfg := Defaults()
	clone := cfg.Clone()
	clone.Server.Address = "x"
	clone.Audio.Input.ChannelMap[0] = 9
	if cfg.Server.Address == "x" {
		t.Fatal("clone mutated original address")
	}
	if cfg.Audio.Input.ChannelMap[0] == 9 {
		t.Fatal("clone mutated original channel map")
	}
}

func TestServerHostPort(t *testing.T) {
	cfg := Defaults()
	cfg.Server.Address = "host"
	cfg.Server.Port = 1
	if got := cfg.ServerHostPort(); got != "host:1" {
		t.Fatalf("got %s", got)
	}
}
