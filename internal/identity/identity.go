package identity

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"os"
	"strings"
	"unicode"
)

// DefaultUsername returns a stable talkiepi-<id> name, matching the esp32-mumble
// pattern of prefixing a device-unique identifier (MAC when available).
func DefaultUsername() string {
	if mac := primaryMAC(); mac != "" {
		return "talkiepi-" + mac
	}
	if h := sanitizeHostname(hostname()); h != "" {
		return "talkiepi-" + h
	}
	return "talkiepi-" + randomHex(6)
}

// hostname is overridable in tests.
var hostname = os.Hostname

// interfaces is overridable in tests.
var interfaces = net.Interfaces

func primaryMAC() string {
	ifaces, err := interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		if len(iface.HardwareAddr) < 6 {
			continue
		}
		// Skip all-zero and locally-administered virtual MACs when possible;
		// still accept the first plausible hardware address.
		mac := iface.HardwareAddr
		allZero := true
		for _, b := range mac {
			if b != 0 {
				allZero = false
				break
			}
		}
		if allZero {
			continue
		}
		return strings.ToUpper(hex.EncodeToString(mac[:6]))
	}
	return ""
}

func sanitizeHostname(h string, err error) string {
	if err != nil {
		return ""
	}
	h = strings.TrimSpace(strings.ToLower(h))
	if i := strings.IndexByte(h, '.'); i > 0 {
		h = h[:i]
	}
	var b strings.Builder
	for _, r := range h {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" || out == "localhost" || out == "local" {
		return ""
	}
	if len(out) > 24 {
		out = out[:24]
	}
	return out
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "device"
	}
	return hex.EncodeToString(buf)
}
