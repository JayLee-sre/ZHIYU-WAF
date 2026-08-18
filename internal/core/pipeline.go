package core

import (
	"context"
	"fmt"
)

// Pipeline executes stages in a deterministic order and delegates the final
// action exclusively to its RiskEngine. It is safe for concurrent requests as
// stages are expected to keep request-local state in PipelineState.
type Pipeline struct {
	stages []Stage
	risk   RiskEngine
}

// NewPipeline constructs a pipeline with an explicit stage ordering.
func NewPipeline(risk RiskEngine, stages ...Stage) (*Pipeline, error) {
	if risk == nil {
		return nil, fmt.Errorf("risk engine is required")
	}
	for i, stage := range stages {
		if stage == nil {
			return nil, fmt.Errorf("stage %d is nil", i)
		}
		if stage.Name() == "" {
			return nil, fmt.Errorf("stage %d has no name", i)
		}
	}
	return &Pipeline{stages: append([]Stage(nil), stages...), risk: risk}, nil
}

// Stages returns a copy of the configured processing order for diagnostics.
func (p *Pipeline) Stages() []string {
	if p == nil {
		return nil
	}
	out := make([]string, 0, len(p.stages))
	for _, stage := range p.stages {
		out = append(out, stage.Name())
	}
	return out
}

// Process runs stages, preserving cancellation semantics. Stage failures are
// returned to the caller; callers can choose a fail-open response while still
// recording the operational fault separately.
func (p *Pipeline) Process(ctx context.Context, req *RequestContext) (*Decision, error) {
	if p == nil || p.risk == nil {
		return nil, fmt.Errorf("pipeline is not initialized")
	}
	if req == nil {
		return nil, fmt.Errorf("request context is required")
	}
	if req.Metadata == nil {
		req.Metadata = make(map[string]any)
	}
	state := &PipelineState{Metadata: make(map[string]any)}
	for _, stage := range p.stages {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := stage.Process(ctx, req, state); err != nil {
			return nil, fmt.Errorf("stage %s: %w", stage.Name(), err)
		}
	}
	risk := p.risk.Evaluate(ctx, req, state.Detections)
	return &Decision{
		Action:     risk.Action,
		Risk:       risk,
		Detections: append([]Detection(nil), state.Detections...),
		RequestID:  req.RequestID,
	}, nil
}

// FunctionalStage turns a small pure function into a named Stage. It keeps
// adapters for rules, ACL, normalizers, and behavior engines concise.
type FunctionalStage struct {
	StageName string
	Handler   func(ctx Context, req *RequestContext, state *PipelineState) error
}

func (s FunctionalStage) Name() string { return s.StageName }

func (s FunctionalStage) Process(ctx Context, req *RequestContext, state *PipelineState) error {
	if s.Handler == nil {
		return nil
	}
	return s.Handler(ctx, req, state)
}
