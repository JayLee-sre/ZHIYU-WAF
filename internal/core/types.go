// Package core contains the transport-agnostic V2 security domain types.
package core

import (
	"net"
	"net/http"
	"net/url"
	"time"
)

// RequestContext is the normalized request envelope shared by every V2 stage.
// It intentionally has no dependency on a concrete proxy, database, firewall,
// AI provider, or rule implementation.
type RequestContext struct {
	RequestID string
	SiteID    int64
	ClientIP  net.IP

	Method string
	Host   string
	Path   string
	Query  url.Values
	Header http.Header
	Body   []byte

	StartTime time.Time
	Metadata  map[string]any
}

// Clone returns an independently mutable request context. It is used at
// asynchronous boundaries so providers cannot mutate the request pipeline.
func (r *RequestContext) Clone() *RequestContext {
	if r == nil {
		return nil
	}
	clone := *r
	clone.ClientIP = append(net.IP(nil), r.ClientIP...)
	clone.Query = cloneValues(r.Query)
	clone.Header = r.Header.Clone()
	clone.Body = append([]byte(nil), r.Body...)
	clone.Metadata = make(map[string]any, len(r.Metadata))
	for k, v := range r.Metadata {
		clone.Metadata[k] = v
	}
	return &clone
}

func cloneValues(values url.Values) url.Values {
	if values == nil {
		return nil
	}
	out := make(url.Values, len(values))
	for key, items := range values {
		out[key] = append([]string(nil), items...)
	}
	return out
}

// Detection is evidence produced by a rule, behavior signal, or optional
// provider. A detection does not decide whether a request is blocked.
type Detection struct {
	RuleID     string  `json:"rule_id"`
	Name       string  `json:"name"`
	Category   string  `json:"category"`
	Severity   int     `json:"severity"`
	Confidence float64 `json:"confidence"`
	Message    string  `json:"message"`
	Source     string  `json:"source"`
}

// Clamp returns a copy constrained to valid V2 scoring ranges.
func (d Detection) Clamp() Detection {
	if d.Severity < 0 {
		d.Severity = 0
	}
	if d.Severity > 100 {
		d.Severity = 100
	}
	if d.Confidence < 0 {
		d.Confidence = 0
	}
	if d.Confidence > 1 {
		d.Confidence = 1
	}
	return d
}

// Action is the only decision vocabulary used by the V2 pipeline.
type Action string

const (
	ActionAllow     Action = "allow"
	ActionLog       Action = "log"
	ActionRateLimit Action = "rate_limit"
	ActionBlock     Action = "block"
)

// RiskResult is the output of the risk engine after it aggregates evidence.
type RiskResult struct {
	Score   int      `json:"score"`
	Level   string   `json:"level"`
	Action  Action   `json:"action"`
	Reasons []string `json:"reasons"`
}

// Decision is the final V2 pipeline result. Exactly one action is returned for
// a request; stages add evidence through PipelineState rather than deciding
// network side effects themselves.
type Decision struct {
	Action     Action      `json:"action"`
	Risk       RiskResult  `json:"risk"`
	Detections []Detection `json:"detections,omitempty"`
	RequestID  string      `json:"request_id"`
}

// PipelineState stores evidence accumulated by stages in a single request.
type PipelineState struct {
	Detections []Detection
	Metadata   map[string]any
}

// AddDetection accepts partial detector output while enforcing valid scoring
// bounds at the core boundary.
func (s *PipelineState) AddDetection(d Detection) {
	s.Detections = append(s.Detections, d.Clamp())
}

// Stage is a low-coupling processing unit. It may normalize requests or add
// evidence, but it must not mutate the firewall, database, or global config.
type Stage interface {
	Name() string
	Process(ctx Context, req *RequestContext, state *PipelineState) error
}

// Context deliberately aliases the cancellation/deadline contract without
// coupling core stages to concrete application services.
type Context interface {
	Done() <-chan struct{}
	Err() error
}

// RiskEngine converts detections plus contextual metadata into one decision.
type RiskEngine interface {
	Evaluate(ctx Context, req *RequestContext, detections []Detection) RiskResult
}

// Firewall represents the only allowed host firewall boundary for V2.
type Firewall interface {
	BlockIP(ctx Context, ip net.IP, ttl time.Duration, reason string) error
	UnblockIP(ctx Context, ip net.IP) error
	IsBlocked(ctx Context, ip net.IP) (bool, error)
	Sync(ctx Context) error
	Status(ctx Context) FirewallStatus
}

// FirewallStatus lets the UI surface a degraded host firewall without making
// the application-layer WAF unavailable.
type FirewallStatus struct {
	Available bool   `json:"available"`
	Degraded  bool   `json:"degraded"`
	Message   string `json:"message"`
}

// ThreatProvider is optional and must never be required for local protection.
type ThreatProvider interface {
	CheckIP(ctx Context, ip net.IP) (*ThreatResult, error)
}

// ThreatResult is provider-neutral reputation evidence.
type ThreatResult struct {
	Score  int      `json:"score"`
	Labels []string `json:"labels"`
	Source string   `json:"source"`
}

// AIProvider is optional and may only enrich the asynchronous analysis path.
type AIProvider interface {
	Analyze(ctx Context, event *SecurityEvent) (*AIResult, error)
}

// SecurityEvent is the privacy-minimized event shape passed to optional
// providers and persisted by V2 stores.
type SecurityEvent struct {
	RequestID string
	SiteID    int64
	ClientIP  net.IP
	Method    string
	Host      string
	Path      string
	Action    Action
	Risk      RiskResult
	CreatedAt time.Time
}

// AIResult contains non-authoritative enrichment. The risk engine retains
// control of the final action.
type AIResult struct {
	Confidence float64
	Summary    string
	Category   string
}

// Rule describes an independent detector. It is intentionally unable to
// access persistence, firewall, or providers.
type Rule interface {
	ID() string
	Name() string
	Category() string
	Match(ctx *RequestContext) *Detection
}

// RuleProvider loads optional rule definitions without coupling core logic to
// a remote control plane.
type RuleProvider interface {
	ListRules(ctx Context) ([]Rule, error)
}

// EventStore is the minimal V2 persistence contract used by the pipeline.
type EventStore interface {
	Create(ctx Context, event *SecurityEvent) error
	Query(ctx Context, query EventQuery) ([]SecurityEvent, error)
}

// EventQuery is intentionally small and can be implemented by SQLite without
// requiring Elasticsearch or other external services.
type EventQuery struct {
	SiteID int64
	IP     net.IP
	From   time.Time
	To     time.Time
	Limit  int
}
