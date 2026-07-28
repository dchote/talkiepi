package ptt

import (
	"context"
	"testing"
	"time"
)

func TestManualPressRelease(t *testing.T) {
	m := NewManual("rest")
	if m.Name() != "rest" {
		t.Fatalf("Name = %q", m.Name())
	}

	out := make(chan Event, 4)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.Start(ctx, out); err != nil {
		t.Fatal(err)
	}
	if err := m.Press(); err != nil {
		t.Fatal(err)
	}
	if err := m.Release(); err != nil {
		t.Fatal(err)
	}

	ev1 := recv(t, out)
	if !ev1.Pressed || ev1.Source != "rest" {
		t.Fatalf("ev1 = %+v", ev1)
	}
	ev2 := recv(t, out)
	if ev2.Pressed || ev2.Source != "rest" {
		t.Fatalf("ev2 = %+v", ev2)
	}

	if err := m.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := m.Press(); err == nil {
		t.Fatal("expected error after Stop")
	}
}

func TestManualDefaultName(t *testing.T) {
	m := NewManual("")
	if m.Name() != "manual" {
		t.Fatalf("Name = %q", m.Name())
	}
}

func TestMuxMergesSources(t *testing.T) {
	a := NewManual("a")
	b := NewManual("b")
	mux := NewMux(a, b)

	out := make(chan Event, 8)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := mux.Start(ctx, out); err != nil {
		t.Fatal(err)
	}
	if err := a.Press(); err != nil {
		t.Fatal(err)
	}
	if err := b.Press(); err != nil {
		t.Fatal(err)
	}
	if err := a.Release(); err != nil {
		t.Fatal(err)
	}

	got := map[string]int{}
	for i := 0; i < 3; i++ {
		ev := recv(t, out)
		got[ev.Source]++
		if ev.Source == "a" && i == 0 && !ev.Pressed {
			t.Fatalf("unexpected %+v", ev)
		}
	}
	if got["a"] != 2 || got["b"] != 1 {
		t.Fatalf("got counts %#v", got)
	}

	if err := mux.Stop(); err != nil {
		t.Fatal(err)
	}
}

func TestMuxDoubleStart(t *testing.T) {
	m := NewManual("m")
	mux := NewMux(m)
	out := make(chan Event, 1)
	ctx := context.Background()
	if err := mux.Start(ctx, out); err != nil {
		t.Fatal(err)
	}
	defer mux.Stop()
	if err := mux.Start(ctx, out); err == nil {
		t.Fatal("expected double-start error")
	}
}

func TestMuxSources(t *testing.T) {
	a := NewManual("a")
	mux := NewMux(a)
	srcs := mux.Sources()
	if len(srcs) != 1 || srcs[0].Name() != "a" {
		t.Fatalf("Sources = %v", srcs)
	}
}

func TestGPIOStubEmptyChip(t *testing.T) {
	src, err := NewGPIO(GPIOConfig{})
	if err != nil {
		// empty chip may return error or a no-op source
		return
	}
	out := make(chan Event, 1)
	ctx := context.Background()
	if err := src.Start(ctx, out); err != nil {
		// no-op may refuse start — either is fine
		_ = src.Stop()
		return
	}
	_ = src.Stop()
}

func recv(t *testing.T, out <-chan Event) Event {
	t.Helper()
	select {
	case ev := <-out:
		return ev
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
		return Event{}
	}
}
