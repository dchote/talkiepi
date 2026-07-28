package identity

import (
	"net"
	"os"
	"strings"
	"testing"
)

func TestDefaultUsernameFromMAC(t *testing.T) {
	interfaces = func() ([]net.Interface, error) {
		return []net.Interface{{
			Name:         "eth0",
			Flags:        net.FlagUp,
			HardwareAddr: net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
		}}, nil
	}
	t.Cleanup(func() { interfaces = net.Interfaces })

	got := DefaultUsername()
	if got != "talkiepi-AABBCCDDEEFF" {
		t.Fatalf("got %q", got)
	}
}

func TestDefaultUsernameFromHostname(t *testing.T) {
	interfaces = func() ([]net.Interface, error) { return nil, nil }
	hostname = func() (string, error) { return "kitchen-pi.local", nil }
	t.Cleanup(func() {
		interfaces = net.Interfaces
		hostname = os.Hostname
	})

	got := DefaultUsername()
	if got != "talkiepi-kitchen-pi" {
		t.Fatalf("got %q", got)
	}
}

func TestSanitizeHostname(t *testing.T) {
	if got := sanitizeHostname("Foo_Bar.lan", nil); got != "foo_bar" {
		t.Fatalf("got %q", got)
	}
	if got := sanitizeHostname("localhost", nil); got != "" {
		t.Fatalf("localhost should be empty, got %q", got)
	}
}

func TestDefaultUsernameFallbackPrefix(t *testing.T) {
	interfaces = func() ([]net.Interface, error) { return nil, nil }
	hostname = func() (string, error) { return "localhost", nil }
	t.Cleanup(func() {
		interfaces = net.Interfaces
		hostname = os.Hostname
	})
	got := DefaultUsername()
	if !strings.HasPrefix(got, "talkiepi-") {
		t.Fatalf("got %q", got)
	}
}
