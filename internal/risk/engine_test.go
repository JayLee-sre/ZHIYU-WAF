package risk

import (
	"context"
	"testing"

	"zhiyuwaf/internal/core"
)

func TestEngineUsesStrongestDetectionPerCategory(t *testing.T) {
	engine := New(Config{LogThreshold: 30, RateLimitThreshold: 60, BlockThreshold: 85})
	result := engine.Evaluate(context.Background(), nil, []core.Detection{
		{RuleID: "SQLI-1", Category: "sqli", Severity: 95, Confidence: 1, Message: "sqli"},
		{RuleID: "SQLI-2", Category: "sqli", Severity: 50, Confidence: 1, Message: "sqli-low"},
		{RuleID: "SCAN-1", Category: "scanner", Severity: 20, Confidence: 1, Message: "scanner"},
	})
	if result.Score != 100 || result.Action != core.ActionBlock || result.Level != "critical" {
		t.Fatalf("unexpected risk result: %#v", result)
	}
	if len(result.Reasons) != 3 {
		t.Fatalf("expected reasons to preserve evidence, got %#v", result.Reasons)
	}
}

func TestEngineThresholds(t *testing.T) {
	engine := New(Config{})
	cases := []struct {
		severity int
		want     core.Action
	}{
		{0, core.ActionAllow},
		{30, core.ActionLog},
		{60, core.ActionRateLimit},
		{85, core.ActionBlock},
	}
	for _, tc := range cases {
		result := engine.Evaluate(context.Background(), nil, []core.Detection{{Category: "test", Severity: tc.severity, Confidence: 1}})
		if result.Action != tc.want {
			t.Fatalf("severity %d got %s, want %s", tc.severity, result.Action, tc.want)
		}
	}
}
