// Package firewall contains shared V2 firewall implementations.
package firewall

import (
	"fmt"
	"net"
	"time"

	"zhiyuwaf/internal/core"
)

// Noop is an explicit degraded Firewall implementation for unprivileged,
// non-Linux, development, or unavailable-nftables environments. It never
// pretends that an address is kernel-blocked.
type Noop struct {
	message string
}

func NewNoop(message string) *Noop {
	if message == "" {
		message = "nftables firewall is not available"
	}
	return &Noop{message: message}
}

func (n *Noop) BlockIP(ctx core.Context, ip net.IP, _ time.Duration, _ string) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if ip == nil || ip.To16() == nil {
		return fmt.Errorf("invalid IP address")
	}
	return fmt.Errorf("%s", n.message)
}

func (n *Noop) UnblockIP(ctx core.Context, ip net.IP) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if ip == nil || ip.To16() == nil {
		return fmt.Errorf("invalid IP address")
	}
	return fmt.Errorf("%s", n.message)
}

func (n *Noop) IsBlocked(ctx core.Context, ip net.IP) (bool, error) {
	if ctx != nil && ctx.Err() != nil {
		return false, ctx.Err()
	}
	if ip == nil || ip.To16() == nil {
		return false, fmt.Errorf("invalid IP address")
	}
	return false, nil
}

func (n *Noop) Sync(ctx core.Context) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return fmt.Errorf("%s", n.message)
}

func (n *Noop) Status(_ core.Context) core.FirewallStatus {
	return core.FirewallStatus{Degraded: true, Message: n.message}
}
