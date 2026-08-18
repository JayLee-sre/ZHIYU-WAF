// Package clientip resolves client addresses with an explicit trusted-proxy
// boundary. Forwarded headers are ignored unless the direct peer is trusted.
package clientip

import (
	"net"
	"net/http"
	"strings"
)

// Resolver provides deterministic client IP extraction for the V2 Pipeline.
type Resolver struct {
	trusted []*net.IPNet
}

// New parses exact IPs and CIDRs. Invalid items are rejected instead of being
// silently ignored, making proxy trust configuration auditable.
func New(trusted []string) (*Resolver, error) {
	resolver := &Resolver{trusted: make([]*net.IPNet, 0, len(trusted))}
	for _, raw := range trusted {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if ip := net.ParseIP(raw); ip != nil {
			bits := 128
			if ip.To4() != nil {
				bits = 32
				ip = ip.To4()
			}
			resolver.trusted = append(resolver.trusted, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
			continue
		}
		_, network, err := net.ParseCIDR(raw)
		if err != nil {
			return nil, err
		}
		resolver.trusted = append(resolver.trusted, network)
	}
	return resolver, nil
}

// Resolve returns the direct peer unless that peer is configured as trusted.
// For a trusted peer, the first syntactically valid X-Forwarded-For value is
// preferred, then X-Real-IP. This matches common reverse proxy behavior while
// keeping arbitrary public clients from selecting an identity.
func (r *Resolver) Resolve(request *http.Request) net.IP {
	if request == nil {
		return nil
	}
	peer := parseRemoteAddr(request.RemoteAddr)
	if peer == nil || !r.isTrusted(peer) {
		return peer
	}
	for _, item := range strings.Split(request.Header.Get("X-Forwarded-For"), ",") {
		if ip := net.ParseIP(strings.TrimSpace(item)); ip != nil {
			return ip
		}
	}
	if ip := net.ParseIP(strings.TrimSpace(request.Header.Get("X-Real-IP"))); ip != nil {
		return ip
	}
	return peer
}

func (r *Resolver) isTrusted(ip net.IP) bool {
	for _, network := range r.trusted {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func parseRemoteAddr(raw string) net.IP {
	host, _, err := net.SplitHostPort(strings.TrimSpace(raw))
	if err != nil {
		host = raw
	}
	return net.ParseIP(strings.TrimSpace(host))
}
