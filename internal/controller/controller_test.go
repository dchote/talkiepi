package controller

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/dchote/talkiepi/internal/config"
)

func TestStatusDefaults(t *testing.T) {
	cfg := config.Defaults()
	cfg.SetPath(filepath.Join(t.TempDir(), "t.yaml"))
	c := New(Options{Config: cfg, Version: "0.0.1-test"})
	st := c.Status()
	if st.Version != "0.0.1-test" {
		t.Fatalf("version=%s", st.Version)
	}
	if st.Connected {
		t.Fatal("should not be connected")
	}
	if st.Mode != "push_to_talk" {
		t.Fatalf("mode=%s", st.Mode)
	}
}

func TestApplyConfigPatch(t *testing.T) {
	cfg := config.Defaults()
	cfg.SetPath(filepath.Join(t.TempDir(), "t.yaml"))
	c := New(Options{Config: cfg})
	if err := c.ApplyConfig([]byte("voice:\n  mode: communicator\n")); err != nil {
		t.Fatal(err)
	}
	if c.Config().Clone().Voice.Mode != "communicator" {
		t.Fatal(c.Config().Clone().Voice.Mode)
	}
}

func TestSubscribeUnsubscribe(t *testing.T) {
	cfg := config.Defaults()
	c := New(Options{Config: cfg})
	ch := c.Subscribe()
	c.emit(map[string]any{"type": "ping"})
	select {
	case ev := <-ch:
		if ev.(map[string]any)["type"] != "ping" {
			t.Fatalf("ev=%v", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
	c.Unsubscribe(ch)
}

func TestPTTPushToTalk(t *testing.T) {
	cfg := config.Defaults()
	cfg.Voice.Mode = "push_to_talk"
	c := New(Options{Config: cfg})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := c.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer c.Stop()

	if err := c.ManualPTT().Press(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if c.Status().Transmitting {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !c.Status().Transmitting {
		t.Fatal("expected transmitting")
	}
	_ = c.ManualPTT().Release()
}

func TestCommunicatorToggle(t *testing.T) {
	cfg := config.Defaults()
	cfg.Voice.Mode = "communicator"
	c := New(Options{Config: cfg})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := c.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer c.Stop()
	_ = c.ManualPTT().Press()
	time.Sleep(50 * time.Millisecond)
	if c.Communicator().State().String() == "idle" {
		t.Fatal("expected non-idle after toggle")
	}
}
