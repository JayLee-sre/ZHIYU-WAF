// Package nftables implements the V2 Firewall interface through Netlink.
// It never invokes the nft command or changes tables outside table inet zhiyu.
package nftables

import (
	"fmt"
	"net"
	"sync"
	"time"

	nft "github.com/google/nftables"
	"github.com/google/nftables/expr"
	"golang.org/x/sys/unix"

	"zhiyuwaf/internal/core"
)

const (
	tableName = "zhiyu"
	chainName = "input"
	setIPv4   = "blocked_ipv4"
	setIPv6   = "blocked_ipv6"
)

// Manager owns only ZHIYU's nftables table. It never flushes the ruleset,
// firewalld tables, Docker rules, or user-managed tables.
type Manager struct {
	mu      sync.Mutex
	factory func() (*nft.Conn, error)
	table   *nft.Table
	ipv4    *nft.Set
	ipv6    *nft.Set
	blocked map[string]time.Time
	status  core.FirewallStatus
}

// New returns a lazy manager. Netlink is opened only when a firewall operation
// is requested, allowing the WAF's application-level protection to work on
// hosts where nftables is unavailable or the process lacks CAP_NET_ADMIN.
func New() *Manager {
	return &Manager{
		factory: func() (*nft.Conn, error) { return nft.New() },
		blocked: make(map[string]time.Time),
		status: core.FirewallStatus{
			Available: false,
			Degraded:  true,
			Message:   "nftables not initialized",
		},
	}
}

// BlockIP adds a validated IPv4 or IPv6 address to a timeout-capable set.
// A zero ttl represents a persistent set element.
func (m *Manager) BlockIP(ctx core.Context, ip net.IP, ttl time.Duration, _ string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	key, family, err := normalizedIP(ip)
	if err != nil {
		return err
	}
	if ttl < 0 {
		return fmt.Errorf("block ttl must not be negative")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.ensureLocked(); err != nil {
		return err
	}
	set := m.ipv6
	if family == 4 {
		set = m.ipv4
	}
	conn, err := m.factory()
	if err != nil {
		return m.degradeLocked(fmt.Errorf("open nftables connection: %w", err))
	}
	element := nft.SetElement{Key: []byte(key), Timeout: ttl}
	if err := conn.SetAddElements(set, []nft.SetElement{element}); err != nil {
		return m.degradeLocked(fmt.Errorf("queue nftables block: %w", err))
	}
	if err := conn.Flush(); err != nil {
		return m.degradeLocked(fmt.Errorf("apply nftables block: %w", err))
	}
	expires := time.Time{}
	if ttl > 0 {
		expires = time.Now().Add(ttl)
	}
	m.blocked[ip.String()] = expires
	m.status = core.FirewallStatus{Available: true, Message: "nftables active"}
	return nil
}

// UnblockIP removes one address from ZHIYU's own set. It does not alter any
// unrelated rule or set.
func (m *Manager) UnblockIP(ctx core.Context, ip net.IP) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	key, family, err := normalizedIP(ip)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.ensureLocked(); err != nil {
		return err
	}
	set := m.ipv6
	if family == 4 {
		set = m.ipv4
	}
	conn, err := m.factory()
	if err != nil {
		return m.degradeLocked(fmt.Errorf("open nftables connection: %w", err))
	}
	if err := conn.SetDeleteElements(set, []nft.SetElement{{Key: []byte(key)}}); err != nil {
		return m.degradeLocked(fmt.Errorf("queue nftables unblock: %w", err))
	}
	if err := conn.Flush(); err != nil {
		return m.degradeLocked(fmt.Errorf("apply nftables unblock: %w", err))
	}
	delete(m.blocked, ip.String())
	m.status = core.FirewallStatus{Available: true, Message: "nftables active"}
	return nil
}

// IsBlocked returns the locally synchronized state. Expired temporary entries
// are dropped from the local cache; the kernel remains the enforcement source.
func (m *Manager) IsBlocked(ctx core.Context, ip net.IP) (bool, error) {
	if err := contextErr(ctx); err != nil {
		return false, err
	}
	if _, _, err := normalizedIP(ip); err != nil {
		return false, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	until, ok := m.blocked[ip.String()]
	if !ok {
		return false, nil
	}
	if !until.IsZero() && time.Now().After(until) {
		delete(m.blocked, ip.String())
		return false, nil
	}
	return true, nil
}

// Sync verifies that the namespace and sets are available. Persistent blocklist
// replay is intentionally owned by the storage/blocklist service so this
// Firewall remains independent of SQLite.
func (m *Manager) Sync(ctx core.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ensureLocked()
}

func (m *Manager) Status(_ core.Context) core.FirewallStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status
}

func (m *Manager) ensureLocked() error {
	if m.table != nil && m.ipv4 != nil && m.ipv6 != nil {
		return nil
	}
	conn, err := m.factory()
	if err != nil {
		return m.degradeLocked(fmt.Errorf("open nftables connection: %w", err))
	}
	table, err := conn.ListTableOfFamily(tableName, nft.TableFamilyINet)
	if err == nil && table != nil {
		ipv4, err4 := conn.GetSetByName(table, setIPv4)
		ipv6, err6 := conn.GetSetByName(table, setIPv6)
		if err4 == nil && err6 == nil {
			m.table, m.ipv4, m.ipv6 = table, ipv4, ipv6
			m.status = core.FirewallStatus{Available: true, Message: "nftables active"}
			return nil
		}
		return m.degradeLocked(fmt.Errorf("zhiyu nftables table exists but required sets are missing"))
	}

	table = conn.AddTable(&nft.Table{Family: nft.TableFamilyINet, Name: tableName})
	chain := conn.AddChain(&nft.Chain{
		Name:     chainName,
		Table:    table,
		Type:     nft.ChainTypeFilter,
		Hooknum:  nft.ChainHookInput,
		Priority: nft.ChainPriorityFilter,
	})
	ipv4 := &nft.Set{Table: table, Name: setIPv4, KeyType: nft.TypeIPAddr, HasTimeout: true, Comment: "ZHIYU-WAF IPv4 blocklist"}
	ipv6 := &nft.Set{Table: table, Name: setIPv6, KeyType: nft.TypeIP6Addr, HasTimeout: true, Comment: "ZHIYU-WAF IPv6 blocklist"}
	if err := conn.AddSet(ipv4, nil); err != nil {
		return m.degradeLocked(fmt.Errorf("create IPv4 blocklist set: %w", err))
	}
	if err := conn.AddSet(ipv6, nil); err != nil {
		return m.degradeLocked(fmt.Errorf("create IPv6 blocklist set: %w", err))
	}
	conn.AddRule(&nft.Rule{Table: table, Chain: chain, Exprs: ipv4DropExpressions(ipv4)})
	conn.AddRule(&nft.Rule{Table: table, Chain: chain, Exprs: ipv6DropExpressions(ipv6)})
	if err := conn.Flush(); err != nil {
		return m.degradeLocked(fmt.Errorf("initialize zhiyu nftables table: %w", err))
	}
	m.table, m.ipv4, m.ipv6 = table, ipv4, ipv6
	m.status = core.FirewallStatus{Available: true, Message: "nftables active"}
	return nil
}

func ipv4DropExpressions(set *nft.Set) []expr.Any {
	return []expr.Any{
		&expr.Meta{Key: expr.MetaKeyNFPROTO, Register: 1},
		&expr.Cmp{Op: expr.CmpOpEq, Register: 1, Data: []byte{unix.NFPROTO_IPV4}},
		&expr.Payload{DestRegister: 1, Base: expr.PayloadBaseNetworkHeader, Offset: 12, Len: 4},
		&expr.Lookup{SourceRegister: 1, SetName: set.Name, SetID: set.ID},
		&expr.Verdict{Kind: expr.VerdictDrop},
	}
}

func ipv6DropExpressions(set *nft.Set) []expr.Any {
	return []expr.Any{
		&expr.Meta{Key: expr.MetaKeyNFPROTO, Register: 1},
		&expr.Cmp{Op: expr.CmpOpEq, Register: 1, Data: []byte{unix.NFPROTO_IPV6}},
		&expr.Payload{DestRegister: 1, Base: expr.PayloadBaseNetworkHeader, Offset: 8, Len: 16},
		&expr.Lookup{SourceRegister: 1, SetName: set.Name, SetID: set.ID},
		&expr.Verdict{Kind: expr.VerdictDrop},
	}
}

func (m *Manager) degradeLocked(err error) error {
	m.status = core.FirewallStatus{Degraded: true, Message: err.Error()}
	return err
}

func normalizedIP(ip net.IP) ([]byte, int, error) {
	if v4 := ip.To4(); v4 != nil {
		return append([]byte(nil), v4...), 4, nil
	}
	if v6 := ip.To16(); v6 != nil {
		return append([]byte(nil), v6...), 6, nil
	}
	return nil, 0, fmt.Errorf("invalid IP address")
}

func contextErr(ctx core.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}
