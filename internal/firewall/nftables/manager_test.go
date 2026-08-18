package nftables

import (
	"net"
	"testing"
)

func TestNormalizedIPSupportsBothFamilies(t *testing.T) {
	v4, family, err := normalizedIP(net.ParseIP("203.0.113.9"))
	if err != nil || family != 4 || len(v4) != 4 {
		t.Fatalf("unexpected IPv4 result bytes=%v family=%d err=%v", v4, family, err)
	}
	v6, family, err := normalizedIP(net.ParseIP("2001:db8::9"))
	if err != nil || family != 6 || len(v6) != 16 {
		t.Fatalf("unexpected IPv6 result bytes=%v family=%d err=%v", v6, family, err)
	}
}

func TestNormalizedIPRejectsNil(t *testing.T) {
	if _, _, err := normalizedIP(nil); err == nil {
		t.Fatal("expected nil IP to be rejected")
	}
}

func TestNewStartsDegradedUntilSync(t *testing.T) {
	manager := New()
	status := manager.Status(nil)
	if !status.Degraded || status.Available {
		t.Fatalf("unexpected initial status: %#v", status)
	}
}
