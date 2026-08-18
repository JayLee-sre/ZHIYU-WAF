package core

import (
	"context"
	"errors"
	"net"
	"net/url"
	"reflect"
	"testing"
	"time"
)

type testRiskEngine struct{}

func (testRiskEngine) Evaluate(_ Context, _ *RequestContext, detections []Detection) RiskResult {
	if len(detections) == 0 {
		return RiskResult{Action: ActionAllow, Level: "low"}
	}
	return RiskResult{Score: 90, Level: "critical", Action: ActionBlock, Reasons: []string{"test"}}
}

func TestPipelineProcessesStagesInOrderAndDecidesOnce(t *testing.T) {
	var order []string
	first := FunctionalStage{StageName: "first", Handler: func(_ Context, _ *RequestContext, state *PipelineState) error {
		order = append(order, "first")
		state.AddDetection(Detection{RuleID: "R-1", Severity: 120, Confidence: 2})
		return nil
	}}
	second := FunctionalStage{StageName: "second", Handler: func(_ Context, _ *RequestContext, state *PipelineState) error {
		order = append(order, "second")
		state.AddDetection(Detection{RuleID: "R-2", Severity: 50, Confidence: 0.8})
		return nil
	}}
	pipeline, err := NewPipeline(testRiskEngine{}, first, second)
	if err != nil {
		t.Fatalf("NewPipeline: %v", err)
	}
	request := &RequestContext{RequestID: "req-1", ClientIP: net.ParseIP("203.0.113.9"), Query: url.Values{}}
	decision, err := pipeline.Process(context.Background(), request)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if !reflect.DeepEqual(order, []string{"first", "second"}) {
		t.Fatalf("unexpected stage order: %v", order)
	}
	if decision.Action != ActionBlock || decision.Risk.Score != 90 {
		t.Fatalf("unexpected decision: %#v", decision)
	}
	if decision.Detections[0].Severity != 100 || decision.Detections[0].Confidence != 1 {
		t.Fatalf("detection was not clamped: %#v", decision.Detections[0])
	}
}

func TestPipelineStopsOnStageError(t *testing.T) {
	pipeline, err := NewPipeline(testRiskEngine{}, FunctionalStage{StageName: "broken", Handler: func(_ Context, _ *RequestContext, _ *PipelineState) error {
		return errors.New("broken stage")
	}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = pipeline.Process(context.Background(), &RequestContext{})
	if err == nil || err.Error() != "stage broken: broken stage" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequestContextCloneIsIndependent(t *testing.T) {
	request := &RequestContext{
		ClientIP:  net.ParseIP("2001:db8::1"),
		Query:     url.Values{"a": {"one"}},
		Metadata:  map[string]any{"x": "one"},
		Body:      []byte("body"),
		StartTime: time.Now(),
	}
	clone := request.Clone()
	clone.Query.Set("a", "two")
	clone.Metadata["x"] = "two"
	clone.Body[0] = 'B'
	if request.Query.Get("a") != "one" || request.Metadata["x"] != "one" || string(request.Body) != "body" {
		t.Fatalf("clone mutated original: %#v", request)
	}
}
