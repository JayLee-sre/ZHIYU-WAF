package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"zhiyuwaf/internal/clientip"
	"zhiyuwaf/internal/core"
	"zhiyuwaf/internal/engine"
	"zhiyuwaf/internal/model"
	v2service "zhiyuwaf/internal/v2"
)

const maxInspectableBodyBytes int64 = 2 << 20 // 2 MiB

type handlerConfig struct {
	backendAddr    string
	readTimeout    time.Duration
	writeTimeout   time.Duration
	dynamicProtect bool
}

type Handler struct {
	mu               sync.RWMutex
	cfg              handlerConfig
	pipeline         *engine.Pipeline
	siteResolver     SiteResolver
	sharedTransport  *http.Transport
	onRequest        func() // optional metrics callback
	onBlocked        func() // optional metrics callback
	v2Service        *v2service.Service
	onV2Decision     func(*core.Decision, *core.RequestContext)
	clientIPResolver *clientip.Resolver
}

type SiteRoute struct {
	ID               string
	Name             string
	Domain           string
	Upstream         string
	AIEnabled        bool
	ChallengeEnabled bool
	MaintenanceMode  bool
	SiteType         string
	Mode             string
}

type SiteResolver interface {
	ResolveSite(host string) (*SiteRoute, bool)
}

func NewHandler(backendAddr string, pipeline *engine.Pipeline, readTimeout, writeTimeout int) *Handler {
	return &Handler{
		cfg: handlerConfig{
			backendAddr:  backendAddr,
			readTimeout:  time.Duration(readTimeout) * time.Second,
			writeTimeout: time.Duration(writeTimeout) * time.Second,
		},
		pipeline: pipeline,
		sharedTransport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          200,
			MaxIdleConnsPerHost:   20,
			IdleConnTimeout:       90 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
		},
	}
}

func (h *Handler) SetSiteResolver(resolver SiteResolver) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.siteResolver = resolver
}

func (h *Handler) SetDynamicProtect(enabled bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cfg.dynamicProtect = enabled
}

// SetV2Service enables the V2 decision pipeline. When not set, the handler
// retains the legacy pipeline for a controlled migration path.
// SetTrustedProxies configures the only peers allowed to supply client IP
// forwarding headers. Invalid config is logged and falls back to peer IPs.
func (h *Handler) SetTrustedProxies(proxies []string) {
	resolver, err := clientip.New(proxies)
	if err != nil {
		log.Printf("invalid trusted proxy configuration: %v", err)
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clientIPResolver = resolver
}

func (h *Handler) SetV2Service(service *v2service.Service) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.v2Service = service
}

// SetV2DecisionCallback receives completed V2 decisions asynchronously from
// the request path. It is intended for event persistence and optional firewall
// enforcement, never for additional request-time detection.
func (h *Handler) SetV2DecisionCallback(callback func(*core.Decision, *core.RequestContext)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onV2Decision = callback
}

func (h *Handler) SetMetricsCallbacks(onRequest, onBlocked func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onRequest = onRequest
	h.onBlocked = onBlocked
}

// UpdateConfig updates mutable handler fields after a config hot-reload.
func (h *Handler) UpdateConfig(backendAddr string, readTimeout, writeTimeout int, dynamicProtect bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cfg.backendAddr = backendAddr
	h.cfg.readTimeout = time.Duration(readTimeout) * time.Second
	h.cfg.writeTimeout = time.Duration(writeTimeout) * time.Second
	h.cfg.dynamicProtect = dynamicProtect
}

func (h *Handler) getConfig() handlerConfig {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.cfg
}

// ServeHTTP implements http.Handler for HTTP/1.x and HTTP/2 support.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("proxy panic recovered: %v", rec)
			http.Error(w, "Internal Server Error", 500)
		}
	}()

	// Handle CONNECT for HTTPS tunneling
	if r.Method == http.MethodConnect {
		cfg := h.getConfig()
		route := h.resolveSite(r.Host)
		allowed := false
		if route != nil && route.Upstream != "" && r.Host == route.Upstream {
			allowed = true
		}
		if r.Host == cfg.backendAddr {
			allowed = true
		}
		if !allowed {
			http.Error(w, "CONNECT not allowed", http.StatusForbidden)
			return
		}
		h.handleTunnelHTTP(w, r)
		return
	}

	// Health check — always 200, no auth, before everything
	if r.URL.Path == "/healthz" && r.Method == "GET" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"status":"ok"}`))
		return
	}

	// Handle verification endpoint BEFORE pipeline inspection
	if r.URL.Path == "/__zhiyu_waf_verify" && r.Method == "POST" {
		h.handleVerifyHTTP(w, r)
		return
	}
	if r.URL.Path == "/__zhiyu_waf_logo.png" && r.Method == "GET" {
		h.serveLogoHTTP(w, r)
		return
	}

	cfg := h.getConfig()

	// Track request metrics (snapshot callbacks under lock to avoid data race)
	h.mu.RLock()
	onRequest := h.onRequest
	onBlocked := h.onBlocked
	h.mu.RUnlock()
	if onRequest != nil {
		onRequest()
	}

	// Build parsed request for inspection
	route := h.resolveSite(r.Host)
	if route != nil && route.MaintenanceMode {
		h.serveMaintenanceHTTP(w, route)
		return
	}
	parsed, err := h.buildParsedRequest(r, cfg.readTimeout)
	if err != nil {
		http.Error(w, "Request Entity Too Large", http.StatusRequestEntityTooLarge)
		return
	}
	if route != nil {
		parsed.SiteID = route.ID
		parsed.SiteName = route.Name
		parsed.Domain = route.Domain
		parsed.SkipAI = !route.AIEnabled
	}
	if parsed.Domain == "" {
		parsed.Domain = normalizeHost(r.Host)
	}

	// Run detection pipeline
	ctx, cancel := context.WithTimeout(r.Context(), cfg.readTimeout)
	defer cancel()

	h.mu.RLock()
	v2Pipeline := h.v2Service
	onV2Decision := h.onV2Decision
	h.mu.RUnlock()
	if v2Pipeline != nil {
		decision, normalized, err := v2Pipeline.Inspect(ctx, parsed)
		if err != nil {
			// V2 follows the local fail-open operational principle: a pipeline
			// fault is visible in logs but never makes the protected site fail.
			log.Printf("V2 pipeline error: %v", err)
		} else if decision != nil {
			monitorOnly := route != nil && route.Mode == "monitor"
			observeOnly := h.pipeline.IsObservationMode() || monitorOnly

			// In observation mode, persist the evidence but never hand a Block or
			// RateLimit action to the persistence callback. The callback owns
			// nftables escalation, so forwarding the original action would turn a
			// monitor-only request into a kernel-level IP block.
			decisionForPersistence := observationPersistenceDecision(decision, observeOnly)
			if onV2Decision != nil {
				go onV2Decision(decisionForPersistence, normalized)
			}

			switch decision.Action {
			case core.ActionBlock:
				if observeOnly {
					log.Printf("OBSERVE V2 %s risk=%d (would block; mode=%s)", decision.RequestID, decision.Risk.Score, routeMode(route))

				} else {
					if onBlocked != nil {
						onBlocked()
					}
					h.serveBlockedHTTP(w, legacyDecisionResult(decision))
					return
				}
			case core.ActionRateLimit:
				if observeOnly {
					log.Printf("OBSERVE V2 %s risk=%d (would rate limit; mode=%s)", decision.RequestID, decision.Risk.Score, routeMode(route))

				} else {
					if onBlocked != nil {
						onBlocked()
					}
					w.Header().Set("Retry-After", "60")
					http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
					return
				}
			}
		}
	} else {
		result := h.pipeline.Inspect(ctx, parsed)
		if result != nil && result.Blocked {
			if h.pipeline.IsObservationMode() {
				// Observation mode: log but don't block
				log.Printf("OBSERVE %s [%s] %s: %s (would block)", result.RuleID, result.Severity, result.RuleName, result.Message)
			} else {
				if onBlocked != nil {
					onBlocked()
				}
				h.serveBlockedHTTP(w, result)
				return
			}
		}
	}

	// Check challenge cookie — skip for static assets
	// Default route: only challenge if dynamicProtect is enabled
	// Named route: challenge if ChallengeEnabled is set
	needChallenge := false
	if route == nil && cfg.dynamicProtect {
		needChallenge = true
	} else if route != nil && route.ChallengeEnabled {
		needChallenge = true
	}
	if needChallenge && !isStaticAsset(r.URL.Path) {
		cookie := getCookieValue(r, "_zhiyu_waf_verified")
		if cookie == "" || !verifyCookie(cookie) {
			h.serveChallengeHTTP(w)
			return
		}
	}

	// Forward to backend
	backendAddr := cfg.backendAddr
	if route != nil && route.Upstream != "" {
		backendAddr = route.Upstream
	}
	h.forwardRequestHTTP(w, r, backendAddr, cfg.dynamicProtect)
}

// forwardRequestHTTP forwards the request to the backend and writes the response.
// If dynamic protection is enabled and the response is HTML, it injects a mutation script.
func routeMode(route *SiteRoute) string {
	if route == nil || route.Mode == "" {
		return "protect"
	}
	return route.Mode
}

func legacyDecisionResult(decision *core.Decision) *engine.DetectionResult {
	result := &engine.DetectionResult{Blocked: true, RuleID: "V2-RISK", RuleName: "V2 Risk Engine", Severity: decision.Risk.Level, Source: "v2", Message: strings.Join(decision.Risk.Reasons, "; ")}
	if len(decision.Detections) > 0 {
		first := decision.Detections[0]
		if first.RuleID != "" {
			result.RuleID = first.RuleID
		}
		if first.Name != "" {
			result.RuleName = first.Name
		}
		if first.Message != "" {
			result.Message = first.Message
		}
	}
	return result
}

func (h *Handler) forwardRequestHTTP(w http.ResponseWriter, r *http.Request, backendAddr string, dynamicProtect bool) {
	// Create a reverse proxy that reuses a shared Transport
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = backendAddr
			req.Host = r.Host
			if origHost := r.Host; origHost != "" {
				req.Header.Set("X-Forwarded-Host", origHost)
			}
			req.Header.Set("X-Forwarded-For", extractRealClientIPFromReq(r))
			proto := "http"
			if r.TLS != nil {
				proto = "https"
			}
			req.Header.Set("X-Forwarded-Proto", proto)
			// When dynamic protection is enabled, strip Accept-Encoding so the
			// backend returns uncompressed HTML that we can safely mutate.
			if dynamicProtect {
				req.Header.Del("Accept-Encoding")
			}
		},
		Transport: h.sharedTransport,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("backend proxy error: %v", err)
			http.Error(w, "Bad Gateway", 502)
		},
	}

	// Dynamic protection: intercept HTML responses
	if dynamicProtect {
		proxy.ModifyResponse = func(resp *http.Response) error {
			ct := resp.Header.Get("Content-Type")
			if !isHTMLContentType(ct) {
				return nil
			}
			body, err := io.ReadAll(io.LimitReader(resp.Body, maxInspectableBodyBytes+1))
			if err != nil {
				return err
			}
			resp.Body.Close()
			if int64(len(body)) > maxInspectableBodyBytes {
				resp.Body = io.NopCloser(bytes.NewReader(body))
				return nil
			}

			modified := injectDynamicScript(body)
			resp.Body = io.NopCloser(bytes.NewReader(modified))
			resp.ContentLength = int64(len(modified))
			resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(modified)))
			resp.Header.Del("Content-Encoding")
			return nil
		}
	}

	proxy.ServeHTTP(w, r)
}

// handleTunnelHTTP handles CONNECT requests via HTTP Hijacker.
func (h *Handler) handleTunnelHTTP(w http.ResponseWriter, r *http.Request) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, bufrw, err := hijacker.Hijack()
	if err != nil {
		log.Printf("hijack failed: %v", err)
		return
	}
	defer clientConn.Close()

	backendConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		log.Printf("tunnel dial %s failed: %v", r.Host, err)
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer backendConn.Close()

	// Respond 200 Connection Established
	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// Bidirectional copy — close connections on first error to unblock the other direction
	errc := make(chan error, 2)
	go func() {
		// Use bufrw as reader to include any buffered data from the hijack
		_, err := io.Copy(backendConn, bufrw)
		errc <- err
	}()
	go func() {
		_, err := io.Copy(clientConn, backendConn)
		errc <- err
	}()

	// Wait for first error, then close both connections to unblock the other goroutine
	<-errc
	clientConn.Close()
	backendConn.Close()
	<-errc // drain the second error
}

func (h *Handler) handleVerifyHTTP(w http.ResponseWriter, r *http.Request) {
	cookie := makeVerifiedCookie()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Set-Cookie", "_zhiyu_waf_verified="+cookie+"; Path=/; Max-Age=86400; HttpOnly")
	w.WriteHeader(200)
	w.Write([]byte(`{"status":"verified"}`))
}

func (h *Handler) serveChallengeHTTP(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(200)
	w.Write([]byte(challengeHTML))
}

func (h *Handler) serveLogoHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(200)
	w.Write([]byte(`<svg viewBox="0 0 64 64" fill="none" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="ZhiYu-WAF"><path d="M32 6L54 18V36C54 48 44 56 32 58C20 56 10 48 10 36V18L32 6Z" fill="url(#g)"/><path d="M23 34L30 41L42 28" stroke="white" stroke-width="6" stroke-linecap="round" stroke-linejoin="round" opacity="0.9"/><defs><linearGradient id="g" x1="10" y1="6" x2="54" y2="58" gradientUnits="userSpaceOnUse"><stop stop-color="#6366F1"/><stop offset="1" stop-color="#A78BFA"/></linearGradient></defs></svg>`))
}

func (h *Handler) serveMaintenanceHTTP(w http.ResponseWriter, route *SiteRoute) {
	siteName := "网站"
	if route != nil && route.Name != "" {
		siteName = route.Name
	}
	siteName = html.EscapeString(siteName)
	body := fmt.Sprintf(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>%s维护中</title>
  <style>
    body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;background:#f8fafc;color:#111827;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","Microsoft YaHei",sans-serif}
    main{max-width:560px;padding:40px 28px;text-align:center}
    .badge{display:inline-flex;padding:6px 12px;border-radius:999px;background:#fff7ed;color:#c2410c;font-size:13px;font-weight:700}
    h1{margin:18px 0 10px;font-size:28px;line-height:1.25}
    p{margin:0;color:#4b5563;font-size:15px;line-height:1.8}
  </style>
</head>
<body>
  <main>
    <div class="badge">ZhiYu WAF 维护模式</div>
    <h1>%s维护中</h1>
    <p>当前站点正在进行安全处置或系统维护，请稍后再访问。</p>
  </main>
</body>
</html>`, siteName, siteName)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusServiceUnavailable)
	w.Write([]byte(body))
}

func (h *Handler) buildParsedRequest(r *http.Request, timeout time.Duration) (*model.ParsedRequest, error) {
	var bodyBytes []byte
	if r.Body != nil {
		limited := io.LimitReader(r.Body, maxInspectableBodyBytes+1)
		var err error
		bodyBytes, err = io.ReadAll(limited)
		if err != nil {
			return nil, err
		}
		if int64(len(bodyBytes)) > maxInspectableBodyBytes {
			return nil, fmt.Errorf("request body exceeds %d bytes", maxInspectableBodyBytes)
		}
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	preview := string(bodyBytes)
	if len(preview) > 500 {
		preview = preview[:500]
	}

	headers := make(map[string][]string)
	for k, v := range r.Header {
		headers[k] = v
	}

	return &model.ParsedRequest{
		ID:          uuid.New().String(),
		Timestamp:   time.Now(),
		ClientIP:    h.resolveClientIP(r),
		Method:      r.Method,
		URL:         r.URL.String(),
		Path:        r.URL.Path,
		QueryParams: r.URL.Query(),
		Headers:     headers,
		Body:        bodyBytes,
		ContentType: r.Header.Get("Content-Type"),
		UserAgent:   r.Header.Get("User-Agent"),
		BodyPreview: preview,
	}, nil
}

func (h *Handler) resolveSite(host string) *SiteRoute {
	h.mu.RLock()
	resolver := h.siteResolver
	h.mu.RUnlock()
	if resolver == nil {
		return nil
	}
	if route, ok := resolver.ResolveSite(host); ok {
		return route
	}
	return nil
}

func (h *Handler) serveBlockedHTTP(w http.ResponseWriter, result *engine.DetectionResult) {
	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"blocked":   true,
		"rule_id":   result.RuleID,
		"rule_name": result.RuleName,
		"severity":  result.Severity,
		"message":   result.Message,
	})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(403)
	w.Write(bodyBytes)
	log.Printf("BLOCKED %s [%s] %s: %s", result.RuleID, result.Severity, result.RuleName, result.Message)
}

// extractRealClientIPFromReq gets the real client IP, checking X-Forwarded-For / X-Real-IP
func (h *Handler) resolveClientIP(r *http.Request) string {
	h.mu.RLock()
	resolver := h.clientIPResolver
	h.mu.RUnlock()
	if resolver != nil {
		if ip := resolver.Resolve(r); ip != nil {
			return ip.String()
		}
	}
	return extractRealClientIPFromReq(r)
}

// observationPersistenceDecision preserves the detection evidence while changing
// enforcement-capable actions to an audit-only action. This prevents monitor and
// global observation modes from reaching the persistence callback's nftables path.
func observationPersistenceDecision(decision *core.Decision, observeOnly bool) *core.Decision {
	if decision == nil || !observeOnly || (decision.Action != core.ActionBlock && decision.Action != core.ActionRateLimit) {
		return decision
	}
	observed := *decision
	observed.Action = core.ActionLog
	return &observed
}

func extractRealClientIPFromReq(r *http.Request) string {
	peerIP := r.RemoteAddr
	if host, _, err := net.SplitHostPort(peerIP); err == nil {
		peerIP = host
	}
	if !isLoopbackIP(peerIP) {
		return peerIP
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := splitCommaList(xff)
		ip := parts[0]
		if ip != "" && !isLoopbackIP(ip) {
			return ip
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		ip := trimSpace(xri)
		if ip != "" && !isLoopbackIP(ip) {
			return ip
		}
	}
	return peerIP
}

func isLoopbackIP(ip string) bool {
	parsed := net.ParseIP(ip)
	return parsed != nil && parsed.IsLoopback()
}

func splitCommaList(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			parts = append(parts, trimSpace(s[start:i]))
			start = i + 1
		}
	}
	return append(parts, trimSpace(s[start:]))
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\r' || s[start] == '\n') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\r' || s[end-1] == '\n') {
		end--
	}
	return s[start:end]
}

func isHTMLContentType(ct string) bool {
	return len(ct) >= 9 && ct[:9] == "text/html"
}
