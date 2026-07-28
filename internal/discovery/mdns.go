package discovery

import (
	"context"
	"fmt"
	"net"
	"sort"
	"time"

	"github.com/grandcat/zeroconf"
)

// ServiceType is the Bonjour/mDNS type advertised by go-mumble-server (and Murmur).
const ServiceType = "_mumble._tcp"

const localDomain = "local."

// Service is a discovered Mumble server on the LAN.
type Service struct {
	Name      string   `json:"name"`
	Host      string   `json:"host"`
	Port      int      `json:"port"`
	Addresses []string `json:"addresses"`
}

// PreferredHost returns the best dial target (IPv4 preferred, else host, else first addr).
func (s Service) PreferredHost() string {
	for _, a := range s.Addresses {
		ip := net.ParseIP(a)
		if ip != nil && ip.To4() != nil {
			return a
		}
	}
	if s.Host != "" {
		return stringsTrimDot(s.Host)
	}
	if len(s.Addresses) > 0 {
		return s.Addresses[0]
	}
	return ""
}

func stringsTrimDot(s string) string {
	for len(s) > 0 && s[len(s)-1] == '.' {
		s = s[:len(s)-1]
	}
	return s
}

// Browser discovers Mumble servers via mDNS.
type Browser interface {
	Browse(ctx context.Context, timeout time.Duration) ([]Service, error)
}

// MDNSBrowser uses grandcat/zeroconf to browse for _mumble._tcp.
type MDNSBrowser struct{}

// NewMDNSBrowser returns a real mDNS browser.
func NewMDNSBrowser() *MDNSBrowser { return &MDNSBrowser{} }

// Browse looks for Mumble services until timeout.
func (b *MDNSBrowser) Browse(ctx context.Context, timeout time.Duration) ([]Service, error) {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, fmt.Errorf("discovery: resolver: %w", err)
	}

	entries := make(chan *zeroconf.ServiceEntry, 16)
	browseCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- resolver.Browse(browseCtx, ServiceType, localDomain, entries)
	}()

	byKey := map[string]Service{}
	for {
		select {
		case <-browseCtx.Done():
			goto done
		case err := <-errCh:
			if err != nil && browseCtx.Err() == nil {
				return nil, fmt.Errorf("discovery: browse: %w", err)
			}
			goto done
		case e, ok := <-entries:
			if !ok {
				goto done
			}
			if e == nil {
				continue
			}
			addrs := make([]string, 0, len(e.AddrIPv4)+len(e.AddrIPv6))
			for _, ip := range e.AddrIPv4 {
				addrs = append(addrs, ip.String())
			}
			for _, ip := range e.AddrIPv6 {
				addrs = append(addrs, ip.String())
			}
			svc := Service{
				Name:      e.Instance,
				Host:      stringsTrimDot(e.HostName),
				Port:      e.Port,
				Addresses: addrs,
			}
			key := fmt.Sprintf("%s|%s|%d", svc.Name, svc.PreferredHost(), svc.Port)
			byKey[key] = svc
		}
	}

done:
	out := make([]Service, 0, len(byKey))
	for _, s := range byKey {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].PreferredHost() < out[j].PreferredHost()
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// StaticBrowser returns a fixed list (tests).
type StaticBrowser struct {
	Services []Service
	Err      error
}

// Browse implements Browser.
func (b StaticBrowser) Browse(context.Context, time.Duration) ([]Service, error) {
	if b.Err != nil {
		return nil, b.Err
	}
	out := append([]Service(nil), b.Services...)
	return out, nil
}
