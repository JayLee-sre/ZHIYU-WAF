package engine

import (
	"context"
	"strings"

	"zhiyuwaf/internal/core"
	"zhiyuwaf/internal/model"
)

// DetectV2 evaluates every matching rule and returns evidence for the V2 risk
// engine. Unlike the legacy Inspect path, it never decides Allow or Block.
func (rs *RuleSet) DetectV2(ctx context.Context, req *model.ParsedRequest) []core.Detection {
	if rs == nil || req == nil {
		return nil
	}
	out := make([]core.Detection, 0)
	for location, rules := range rs.RulesByLocation() {
		if ctx.Err() != nil {
			return out
		}
		texts := ExtractLocationText(location, req)
		for _, rule := range rules {
			if result := matchRulePatterns(rule, texts); result != nil {
				out = append(out, core.Detection{
					RuleID:     result.RuleID,
					Name:       result.RuleName,
					Category:   categoryForRule(result.RuleID),
					Severity:   severityScore(result.Severity),
					Confidence: 1,
					Message:    result.Message,
					Source:     "rule",
				})
			}
		}
	}
	return out
}

func severityScore(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "critical":
		return 95
	case "high":
		return 75
	case "medium":
		return 50
	case "low":
		return 25
	default:
		return 40
	}
}

func categoryForRule(id string) string {
	prefix := strings.ToUpper(strings.SplitN(id, "-", 2)[0])
	switch prefix {
	case "SQLI", "NOSQL":
		return "sqli"
	case "XSS":
		return "xss"
	case "CMDI", "CMD":
		return "command_injection"
	case "TRAVERSAL", "LFI":
		return "path_traversal"
	case "SSRF":
		return "ssrf"
	case "SENSITIVE":
		return "sensitive_file"
	case "SCAN", "SCANNER":
		return "scanner"
	default:
		return "rule"
	}
}
