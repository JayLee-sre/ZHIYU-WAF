package sshmon

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	"zhiyuwaf/internal/core"
	"zhiyuwaf/internal/model"
	"zhiyuwaf/internal/store"
)

type fakeFirewall struct {
	blocked []net.IP
}

func (f *fakeFirewall) BlockIP(_ core.Context, ip net.IP, _ time.Duration, _ string) error {
	f.blocked = append(f.blocked, append(net.IP(nil), ip...))
	return nil
}
func (f *fakeFirewall) UnblockIP(_ core.Context, _ net.IP) error         { return nil }
func (f *fakeFirewall) IsBlocked(_ core.Context, _ net.IP) (bool, error) { return false, nil }
func (f *fakeFirewall) Sync(_ core.Context) error                        { return nil }
func (f *fakeFirewall) Status(_ core.Context) core.FirewallStatus {
	return core.FirewallStatus{Available: true}
}

func TestProcessLineAcceptedLoginDoesNotPanic(t *testing.T) {
	m := New(Config{Enabled: true, MaxFails: 3, BanMinutes: 1}, nil, nil, nil)
	m.processLine("May 15 10:00:00 host sshd[123]: Accepted password for root from 203.0.113.10 port 55222 ssh2")
}

func TestBlockIPSkipsInvalidAndLocalAddresses(t *testing.T) {
	firewall := &fakeFirewall{}
	m := New(Config{Enabled: true, MaxFails: 1, BanMinutes: 1}, nil, nil, firewall)
	m.blockIP("not-an-ip", "")
	m.blockIP("127.0.0.1", "")
	m.blockIP("::1", "")
	if len(firewall.blocked) != 0 {
		t.Fatalf("protected or invalid addresses must not reach firewall: %#v", firewall.blocked)
	}
}

func TestBlockIPUsesFirewallForIPv4AndIPv6(t *testing.T) {
	firewall := &fakeFirewall{}
	m := New(Config{Enabled: true, MaxFails: 1, BanMinutes: 1}, nil, nil, firewall)
	m.blockIP("203.0.113.10", "test")
	m.blockIP("2001:db8::1", "test")
	if len(firewall.blocked) != 2 {
		t.Fatalf("expected both address families to be sent to firewall, got %#v", firewall.blocked)
	}
}

func TestStopIsIdempotent(t *testing.T) {
	m := New(Config{Enabled: true, MaxFails: 1, BanMinutes: 1}, nil, nil, nil)
	m.Stop()
	m.Stop()
}

func TestSuccessfulLoginFromWhitelistIsNotLogged(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()
	if err := s.AddIPEntry(model.IPEntry{ID: "wl-1", IPAddress: "203.0.113.10", ListType: "whitelist"}); err != nil {
		t.Fatalf("add whitelist: %v", err)
	}
	m := New(Config{Enabled: true, MaxFails: 3, BanMinutes: 1}, s, nil, nil)
	m.processLine("May 15 10:00:00 host sshd[123]: Accepted password for root from 203.0.113.10 port 55222 ssh2")
	events, total, err := s.ListSSHEvents(0, 10, "", "", "")
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if total != 0 || len(events) != 0 {
		t.Fatalf("expected no success event for whitelisted IP, got total=%d events=%d", total, len(events))
	}
}

func TestFailedLoginFromWhitelistIsLogged(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()
	if err := s.AddIPEntry(model.IPEntry{ID: "wl-1", IPAddress: "203.0.113.10", ListType: "whitelist"}); err != nil {
		t.Fatalf("add whitelist: %v", err)
	}
	m := New(Config{Enabled: true, MaxFails: 3, BanMinutes: 1}, s, nil, nil)
	m.processLine("May 15 10:00:00 host sshd[123]: Failed password for root from 203.0.113.10 port 55222 ssh2")
	events, total, err := s.ListSSHEvents(0, 10, "", "", "")
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if total != 1 || len(events) != 1 {
		t.Fatalf("expected failed event for whitelisted IP, got total=%d events=%d", total, len(events))
	}
	if events[0].EventType != "failed" {
		t.Fatalf("expected failed event, got %q", events[0].EventType)
	}
}

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.NewStore(filepath.Join(t.TempDir(), "zhiyu-waf.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	return s
}

var _ core.Firewall = (*fakeFirewall)(nil)
var _ = context.Background
