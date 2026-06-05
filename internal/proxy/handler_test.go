package proxy

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"zhiyuwaf/internal/engine"
	"zhiyuwaf/internal/model"
)

func TestExtractRealClientIPIgnoresSpoofedForwardedHeaders(t *testing.T) {
	req := httptestRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "8.8.8.8")
	req.Header.Set("X-Real-IP", "9.9.9.9")
	req.RemoteAddr = "203.0.113.10:12345"

	got := extractRealClientIPFromReq(req)
	if got != "203.0.113.10" {
		t.Fatalf("expected direct peer IP, got %q", got)
	}
}

func TestExtractRealClientIPTrustsLocalProxyHeaders(t *testing.T) {
	req := httptestRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "8.8.8.8, 127.0.0.1")
	req.RemoteAddr = "127.0.0.1:12345"

	got := extractRealClientIPFromReq(req)
	if got != "8.8.8.8" {
		t.Fatalf("expected forwarded client IP, got %q", got)
	}
}

func TestBuildParsedRequestRejectsOversizedBody(t *testing.T) {
	handler := NewHandler("127.0.0.1:80", engine.NewPipeline(engine.NewRuleSet(), 60, 10), 30, 30)
	body := bytes.NewReader(bytes.Repeat([]byte("a"), int(maxInspectableBodyBytes)+1))
	req := httptestRequest("POST", "/upload", body)

	_, err := handler.buildParsedRequest(req, 30*time.Second)
	if err == nil {
		t.Fatal("expected oversized body error")
	}
}

func TestMaintenanceModeReturnsMaintenancePage(t *testing.T) {
	handler := NewHandler("127.0.0.1:80", engine.NewPipeline(engine.NewRuleSet(), 60, 10), 30, 30)
	handler.SetSiteResolver(NewMemorySiteResolver([]model.Site{{
		ID:              "site-1",
		Name:            "测试站点",
		Domains:         []string{"www.example.com"},
		Upstream:        "127.0.0.1:65535",
		Enabled:         true,
		MaintenanceMode: true,
	}}))

	req := httptest.NewRequest("GET", "http://www.example.com/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected maintenance status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("unexpected content type: %q", got)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("维护模式")) {
		t.Fatalf("maintenance page missing expected text: %s", rec.Body.String())
	}
}

func httptestRequest(method, target string, body *bytes.Reader) *http.Request {
	var reqBody io.Reader
	if body != nil {
		reqBody = body
	}
	req, err := http.NewRequest(method, target, reqBody)
	if err != nil {
		panic(err)
	}
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("User-Agent", "test")
	return req
}

func TestTrimSpace(t *testing.T) {
	if got := trimSpace("\t 1.2.3.4 \r\n"); got != "1.2.3.4" {
		t.Fatalf("unexpected trimmed value: %q", got)
	}
}
