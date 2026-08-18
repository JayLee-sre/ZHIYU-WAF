// Package v2 wires V2 domain stages to existing local rule assets.
package v2

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"zhiyuwaf/internal/behavior"
	"zhiyuwaf/internal/core"
	"zhiyuwaf/internal/engine"
	"zhiyuwaf/internal/engine/builtin"
	"zhiyuwaf/internal/model"
	"zhiyuwaf/internal/normalize"
	"zhiyuwaf/internal/risk"
)

// Config controls only local V2 behavior. Providers and host firewall actions
// are intentionally configured outside this service.
type Config struct {
	RequestsPerMinute int
	BurstSize         int
	Behavior          behavior.Config
	Risk              risk.Config
}

// Service owns request-local V2 processing and bounded local state.
type Service struct {
	pipeline *core.Pipeline
	rules    *engine.RuleSet
	behavior *behavior.Engine
	limiter  *builtin.RateLimiter

	listsMu   sync.RWMutex
	whitelist map[string]bool
	blacklist map[string]bool
}

func New(cfg Config, rules *engine.RuleSet) (*Service, error) {
	service := &Service{
		rules:     rules,
		behavior:  behavior.New(cfg.Behavior),
		limiter:   builtin.NewRateLimiter(cfg.RequestsPerMinute, cfg.BurstSize),
		whitelist: make(map[string]bool),
		blacklist: make(map[string]bool),
	}
	pipeline, err := core.NewPipeline(risk.New(cfg.Risk),
		normalize.Normalizer{},
		core.FunctionalStage{StageName: "acl", Handler: service.aclStage},
		core.FunctionalStage{StageName: "rate_limit", Handler: service.rateStage},
		core.FunctionalStage{StageName: "rules", Handler: service.ruleStage},
		core.FunctionalStage{StageName: "behavior", Handler: service.behaviorStage},
	)
	if err != nil {
		service.Close()
		return nil, err
	}
	service.pipeline = pipeline
	return service, nil
}

func (s *Service) Close() {
	if s == nil {
		return
	}
	if s.behavior != nil {
		s.behavior.Close()
	}
	if s.limiter != nil {
		s.limiter.Stop()
	}
}

func (s *Service) UpdateRules(rules *engine.RuleSet) {
	if s == nil {
		return
	}
	s.listsMu.Lock()
	s.rules = rules
	s.listsMu.Unlock()
}

func (s *Service) UpdateIPLists(whitelist, blacklist map[string]bool) {
	if s == nil {
		return
	}
	s.listsMu.Lock()
	defer s.listsMu.Unlock()
	s.whitelist = normalizeList(whitelist)
	s.blacklist = normalizeList(blacklist)
}

// Inspect converts a proxy-neutral ParsedRequest into the V2 core context and
// returns a single transparent decision. It performs no persistence or host
// firewall operation.
func (s *Service) Inspect(ctx context.Context, request *model.ParsedRequest) (*core.Decision, *core.RequestContext, error) {
	if s == nil || s.pipeline == nil {
		return nil, nil, fmt.Errorf("V2 service is not initialized")
	}
	v2Request, err := requestContext(request)
	if err != nil {
		return nil, nil, err
	}
	decision, err := s.pipeline.Process(ctx, v2Request)
	if err != nil {
		return nil, v2Request, err
	}
	now := time.Now()
	s.behavior.Observe(ctx, behavior.Event{IP: v2Request.ClientIP, Type: behavior.EventRequest, CreatedAt: now})
	if len(decision.Detections) > 0 {
		s.behavior.Observe(ctx, behavior.Event{IP: v2Request.ClientIP, Type: behavior.EventAttack, CreatedAt: now})
	}
	return decision, v2Request, nil
}

func (s *Service) aclStage(_ core.Context, request *core.RequestContext, state *core.PipelineState) error {
	ip := canonicalIP(request.ClientIP)
	if ip == "" {
		return fmt.Errorf("client IP is required")
	}
	s.listsMu.RLock()
	whitelisted := s.whitelist[ip]
	blacklisted := s.blacklist[ip]
	s.listsMu.RUnlock()
	if whitelisted {
		state.Metadata["acl_whitelisted"] = true
		return nil
	}
	if blacklisted {
		state.AddDetection(core.Detection{RuleID: "ACL-BLACKLIST", Name: "IP Blacklist", Category: "acl", Severity: 100, Confidence: 1, Message: "client IP is in the local blocklist", Source: "local"})
	}
	return nil
}

func (s *Service) rateStage(_ core.Context, request *core.RequestContext, state *core.PipelineState) error {
	if state.Metadata["acl_whitelisted"] == true {
		return nil
	}
	if s.limiter != nil && !s.limiter.Allow(canonicalIP(request.ClientIP)) {
		state.AddDetection(core.Detection{RuleID: "RATE-IP", Name: "IP Rate Limit", Category: "frequency", Severity: 70, Confidence: 1, Message: "per-IP request rate exceeded", Source: "local"})
	}
	return nil
}

func (s *Service) ruleStage(ctx core.Context, request *core.RequestContext, state *core.PipelineState) error {
	if state.Metadata["acl_whitelisted"] == true {
		return nil
	}
	s.listsMu.RLock()
	rules := s.rules
	s.listsMu.RUnlock()
	if rules == nil {
		return nil
	}
	legacy := parsedRequest(request)
	for _, detection := range rules.DetectV2(contextFrom(ctx), legacy) {
		state.AddDetection(detection)
	}
	return nil
}

func (s *Service) behaviorStage(_ core.Context, request *core.RequestContext, state *core.PipelineState) error {
	if state.Metadata["acl_whitelisted"] == true {
		return nil
	}
	snapshot := s.behavior.Snapshot(context.Background(), request.ClientIP)
	for _, counts := range snapshot.Windows {
		if counts[behavior.EventAttack] >= 5 {
			state.AddDetection(core.Detection{RuleID: "BEHAVIOR-ATTACK", Name: "Repeated attack behavior", Category: "behavior", Severity: 65, Confidence: 0.9, Message: "repeated attack detections in a local time window", Source: "local"})
			break
		}
		if counts[behavior.EventRequest] >= 120 {
			state.AddDetection(core.Detection{RuleID: "BEHAVIOR-FREQUENCY", Name: "Abnormal request behavior", Category: "behavior", Severity: 45, Confidence: 0.8, Message: "abnormal request volume in a local time window", Source: "local"})
			break
		}
	}
	return nil
}

func requestContext(request *model.ParsedRequest) (*core.RequestContext, error) {
	if request == nil {
		return nil, fmt.Errorf("request is required")
	}
	ip := net.ParseIP(strings.TrimSpace(request.ClientIP))
	if ip == nil {
		return nil, fmt.Errorf("invalid client IP")
	}
	query := make(url.Values, len(request.QueryParams))
	for key, values := range request.QueryParams {
		query[key] = append([]string(nil), values...)
	}
	headers := make(http.Header, len(request.Headers))
	for key, values := range request.Headers {
		headers[key] = append([]string(nil), values...)
	}
	return &core.RequestContext{
		RequestID: request.ID,
		ClientIP:  ip,
		Method:    request.Method,
		Host:      request.Domain,
		Path:      request.Path,
		Query:     query,
		Header:    headers,
		Body:      append([]byte(nil), request.Body...),
		StartTime: request.Timestamp,
		Metadata:  make(map[string]any),
	}, nil
}

func parsedRequest(request *core.RequestContext) *model.ParsedRequest {
	query := make(map[string][]string, len(request.Query))
	for key, values := range request.Query {
		query[key] = append([]string(nil), values...)
	}
	headers := make(map[string][]string, len(request.Header))
	for key, values := range request.Header {
		headers[key] = append([]string(nil), values...)
	}
	return &model.ParsedRequest{
		ID:          request.RequestID,
		Timestamp:   request.StartTime,
		ClientIP:    request.ClientIP.String(),
		Method:      request.Method,
		URL:         request.Path,
		Path:        request.Path,
		QueryParams: query,
		Headers:     headers,
		Body:        append([]byte(nil), request.Body...),
		ContentType: request.Header.Get("Content-Type"),
		UserAgent:   request.Header.Get("User-Agent"),
	}
}

func normalizeList(input map[string]bool) map[string]bool {
	out := make(map[string]bool, len(input))
	for raw, enabled := range input {
		if !enabled {
			continue
		}
		if ip := net.ParseIP(strings.TrimSpace(raw)); ip != nil {
			out[canonicalIP(ip)] = true
		}
	}
	return out
}

func canonicalIP(ip net.IP) string {
	if v4 := ip.To4(); v4 != nil {
		return v4.String()
	}
	if v6 := ip.To16(); v6 != nil {
		return v6.String()
	}
	return ""
}

type backgroundContext struct{ core.Context }

func (backgroundContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (backgroundContext) Value(any) any               { return nil }

func contextFrom(ctx core.Context) context.Context {
	if std, ok := ctx.(context.Context); ok {
		return std
	}
	return backgroundContext{Context: ctx}
}
