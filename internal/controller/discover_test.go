package controller

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/dchote/talkiepi/internal/config"
	"github.com/dchote/talkiepi/internal/discovery"
)

func TestMaybeAutoDiscoverPopulatesEmptyAddress(t *testing.T) {
	cfg := config.Defaults()
	cfg.SetPath(filepath.Join(t.TempDir(), "cfg.yaml"))
	cfg.Server.Address = ""
	cfg.Server.AutoDiscover = true

	c := New(Options{
		Config: cfg,
		Browser: discovery.StaticBrowser{Services: []discovery.Service{{
			Name:      "kitchen",
			Host:      "kitchen.local",
			Port:      64739,
			Addresses: []string{"10.0.0.9"},
		}}},
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	c.maybeAutoDiscover(ctx)

	got := c.Config().Clone()
	if got.Server.Address != "10.0.0.9" || got.Server.Port != 64739 {
		t.Fatalf("got address=%q port=%d", got.Server.Address, got.Server.Port)
	}
}

func TestMaybeAutoDiscoverSkipsWhenAddressSet(t *testing.T) {
	cfg := config.Defaults()
	cfg.SetPath(filepath.Join(t.TempDir(), "cfg.yaml"))
	cfg.Server.Address = "manual.example"
	cfg.Server.AutoDiscover = true

	c := New(Options{
		Config: cfg,
		Browser: discovery.StaticBrowser{Services: []discovery.Service{{
			Name: "other", Host: "x", Port: 1, Addresses: []string{"1.2.3.4"},
		}}},
	})
	c.maybeAutoDiscover(context.Background())
	if c.Config().Clone().Server.Address != "manual.example" {
		t.Fatal("should not overwrite configured address")
	}
}
