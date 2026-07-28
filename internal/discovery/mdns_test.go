package discovery

import (
	"context"
	"testing"
	"time"
)

func TestPreferredHostPrefersIPv4(t *testing.T) {
	s := Service{
		Host:      "mumble.local",
		Addresses: []string{"fe80::1", "192.168.1.10"},
		Port:      64738,
	}
	if got := s.PreferredHost(); got != "192.168.1.10" {
		t.Fatalf("got %q", got)
	}
}

func TestPreferredHostFallsBackToHost(t *testing.T) {
	s := Service{Host: "kitchen.local.", Port: 64738}
	if got := s.PreferredHost(); got != "kitchen.local" {
		t.Fatalf("got %q", got)
	}
}

func TestStaticBrowser(t *testing.T) {
	b := StaticBrowser{Services: []Service{{Name: "a", Host: "h", Port: 1}}}
	got, err := b.Browse(context.Background(), time.Millisecond)
	if err != nil || len(got) != 1 || got[0].Name != "a" {
		t.Fatalf("got %+v err=%v", got, err)
	}
}
