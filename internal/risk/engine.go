// Package risk contains the deterministic local V2 risk engine.
package risk

import (
	"sort"
	"strings"

	"zhiyuwaf/internal/core"
)

const (
	defaultLogThreshold       = 30
	defaultRateLimitThreshold = 60
	defaultBlockThreshold     = 85
)

// Config configures local risk thresholds. It has no cloud dependency.
type Config struct {
	LogThreshold       int
	RateLimitThreshold int
	BlockThreshold     int
}

// Engine aggregates independent detector output. It deliberately uses a
// transparent scoring model so administrators can understand why an action
// was chosen and tune it without an opaque external service.
type Engine struct {
	config Config
}

func New(config Config) *Engine {
	if config.LogThreshold <= 0 {
		config.LogThreshold = defaultLogThreshold
	}
	if config.RateLimitThreshold <= 0 {
		config.RateLimitThreshold = defaultRateLimitThreshold
	}
	if config.BlockThreshold <= 0 {
		config.BlockThreshold = defaultBlockThreshold
	}
	if config.RateLimitThreshold < config.LogThreshold {
		config.RateLimitThreshold = config.LogThreshold
	}
	if config.BlockThreshold < config.RateLimitThreshold {
		config.BlockThreshold = config.RateLimitThreshold
	}
	return &Engine{config: config}
}

// Evaluate returns a risk score in [0,100]. Each detection contributes its
// severity weighted by confidence. Evidence is capped per category so repeated
// equivalent signatures cannot inflate a request beyond the scoring model.
func (e *Engine) Evaluate(_ core.Context, _ *core.RequestContext, detections []core.Detection) core.RiskResult {
	if len(detections) == 0 {
		return core.RiskResult{Score: 0, Level: "low", Action: core.ActionAllow}
	}
	categoryScores := make(map[string]int)
	reasons := make([]string, 0, len(detections))
	for _, raw := range detections {
		d := raw.Clamp()
		category := strings.TrimSpace(strings.ToLower(d.Category))
		if category == "" {
			category = "unknown"
		}
		contribution := int(float64(d.Severity) * d.Confidence)
		if contribution == 0 && d.Severity > 0 {
			contribution = 1
		}
		// A category may supply multiple signals, but only the strongest
		// evidence is considered for a single request decision.
		if contribution > categoryScores[category] {
			categoryScores[category] = contribution
		}
		if d.Message != "" {
			reasons = append(reasons, d.Message)
		} else if d.RuleID != "" {
			reasons = append(reasons, d.RuleID)
		}
	}

	score := 0
	for _, value := range categoryScores {
		score += value
	}
	if score > 100 {
		score = 100
	}
	sort.Strings(reasons)
	reasons = compact(reasons)
	result := core.RiskResult{Score: score, Reasons: reasons}
	switch {
	case score >= e.config.BlockThreshold:
		result.Level = "critical"
		result.Action = core.ActionBlock
	case score >= e.config.RateLimitThreshold:
		result.Level = "high"
		result.Action = core.ActionRateLimit
	case score >= e.config.LogThreshold:
		result.Level = "medium"
		result.Action = core.ActionLog
	default:
		result.Level = "low"
		result.Action = core.ActionAllow
	}
	return result
}

func compact(in []string) []string {
	if len(in) < 2 {
		return in
	}
	out := in[:1]
	for _, item := range in[1:] {
		if item != out[len(out)-1] {
			out = append(out, item)
		}
	}
	return out
}
