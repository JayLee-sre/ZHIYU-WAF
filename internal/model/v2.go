package model

import "time"

// V2Site is the local-first site configuration. Mode controls whether the
// risk decision is only observed, actively enforced, or used for emergency
// application-layer blocking.
type V2Site struct {
	ID         int64     `json:"id"`
	Domain     string    `json:"domain"`
	BackendURL string    `json:"backend_url"`
	Mode       string    `json:"mode"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// SecurityEvent is privacy-minimized evidence persisted for Dashboard and API
// use. Complete request bodies are deliberately not part of this model.
type SecurityEvent struct {
	ID         int64     `json:"id"`
	RequestID  string    `json:"request_id"`
	SiteID     int64     `json:"site_id"`
	ClientIP   string    `json:"client_ip"`
	Method     string    `json:"method"`
	Host       string    `json:"host"`
	Path       string    `json:"path"`
	RuleID     string    `json:"rule_id"`
	Category   string    `json:"category"`
	Severity   int       `json:"severity"`
	Confidence float64   `json:"confidence"`
	RiskScore  int       `json:"risk_score"`
	Action     string    `json:"action"`
	UserAgent  string    `json:"user_agent"`
	CreatedAt  time.Time `json:"created_at"`
}

// BlockedIP is a local firewall blocklist item. ExpiresAt is empty for a
// permanent block; the source distinguishes manual, local risk, and provider
// decisions without granting any provider control over the firewall directly.
type BlockedIP struct {
	ID        int64      `json:"id"`
	IP        string     `json:"ip"`
	Family    int        `json:"family"`
	Reason    string     `json:"reason"`
	Score     int        `json:"score"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Permanent bool       `json:"permanent"`
	Source    string     `json:"source"`
	CreatedAt time.Time  `json:"created_at"`
}

// RiskEvent preserves the transparent result of a risk aggregation decision.
type RiskEvent struct {
	ID        int64     `json:"id"`
	ClientIP  string    `json:"client_ip"`
	Score     int       `json:"score"`
	Level     string    `json:"level"`
	Reason    string    `json:"reason"`
	Action    string    `json:"action"`
	CreatedAt time.Time `json:"created_at"`
}

// SystemStatus gives the Dashboard a provider-neutral health summary.
type SystemStatus struct {
	Version          string            `json:"version"`
	UptimeSeconds    int64             `json:"uptime_seconds"`
	FirewallStatus   string            `json:"firewall_status"`
	DatabaseStatus   string            `json:"database_status"`
	ProviderStatuses map[string]string `json:"provider_statuses"`
}
