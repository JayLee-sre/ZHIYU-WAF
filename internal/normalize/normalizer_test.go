package normalize

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"zhiyuwaf/internal/core"
)

func TestNormalizerCanonicalizesEncodedInput(t *testing.T) {
	normalizer := Normalizer{}
	request := &core.RequestContext{
		Method: " post ",
		Host:   "EXAMPLE.COM ",
		Path:   "/a/%252e%252e/admin",
		Query:  url.Values{"Q": {"%2555NION%2520SELECT"}},
		Header: http.Header{"X-Test": {"  VALUE  "}, "Content-Type": {"application/json"}},
		Body:   []byte(`{"query":"%2555NION%2520SELECT"}`),
	}
	state := &core.PipelineState{}
	if err := normalizer.Process(context.Background(), request, state); err != nil {
		t.Fatalf("Process: %v", err)
	}
	if request.Method != "POST" || request.Host != "example.com" || request.Path != "/admin" {
		t.Fatalf("request was not normalized: %#v", request)
	}
	if got := request.Query.Get("q"); got != "union select" {
		t.Fatalf("query got %q", got)
	}
	if got := request.Header.Get("X-Test"); got != "value" {
		t.Fatalf("header got %q", got)
	}
	if got, _ := request.Metadata["normalized_json"].(string); got == "" {
		t.Fatalf("normalized JSON missing: %#v", request.Metadata)
	}
}

func TestNormalizerRejectsOversizeBody(t *testing.T) {
	normalizer := Normalizer{MaxBodyBytes: 4}
	request := &core.RequestContext{Body: []byte("12345")}
	if err := normalizer.Process(context.Background(), request, &core.PipelineState{}); err == nil {
		t.Fatal("expected body size error")
	}
}
