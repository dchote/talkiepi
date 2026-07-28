package mumbleclient

import (
	"net"
	"testing"
	"time"

	"github.com/dchote/go-mumble-server/pkg/mumble/crypto"
)

func TestNextBackoff(t *testing.T) {
	if got := nextBackoff(0); got != ReconnectMinDelay {
		t.Fatalf("zero: got %v want %v", got, ReconnectMinDelay)
	}
	if got := nextBackoff(ReconnectMinDelay); got != 2*time.Second {
		t.Fatalf("1s: got %v", got)
	}
	if got := nextBackoff(16 * time.Second); got != ReconnectMaxDelay {
		t.Fatalf("16s: got %v want %v", got, ReconnectMaxDelay)
	}
	if got := nextBackoff(ReconnectMaxDelay); got != ReconnectMaxDelay {
		t.Fatalf("max: got %v", got)
	}
}

func TestCryptModeFromKey(t *testing.T) {
	cases := []struct {
		key  []byte
		want crypto.Mode
	}{
		{nil, crypto.ModeLite},
		{[]byte{}, crypto.ModeLite},
		{make([]byte, 16), crypto.ModeLegacy},
		{make([]byte, 32), crypto.ModeSecure},
		{make([]byte, 8), crypto.ModeLegacy},
	}
	for _, tc := range cases {
		if got := cryptModeFromKey(tc.key); got != tc.want {
			t.Fatalf("len=%d: got %v want %v", len(tc.key), got, tc.want)
		}
	}
}

func TestTransportString(t *testing.T) {
	if TransportUDP.String() != "udp" || TransportTCP.String() != "tcp" {
		t.Fatal("transport strings")
	}
}

func TestUDPAddrFromTCP(t *testing.T) {
	tcpAddr := &net.TCPAddr{IP: net.ParseIP("203.0.113.5"), Port: 64738}
	addr, err := udpAddrFromTCP(tcpAddr)
	if err != nil {
		t.Fatal(err)
	}
	if addr.IP.String() != "203.0.113.5" || addr.Port != 64738 {
		t.Fatalf("got %v", addr)
	}
	// Must copy IP so later mutation of TCPAddr cannot affect UDP peer.
	orig := addr.IP.String()
	tcpAddr.IP[len(tcpAddr.IP)-1] ^= 0xff
	if addr.IP.String() != orig {
		t.Fatalf("UDP addr aliased TCP IP: %v", addr.IP)
	}
}
