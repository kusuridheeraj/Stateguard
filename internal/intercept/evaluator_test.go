package intercept

import (
	"context"
	"testing"

	"github.com/kusuridheeraj/stateguard/internal/orchestrator"
)

func TestEvaluateComposeOperation(t *testing.T) {
	evaluator := Evaluator{
		Mode: "fail-closed",
		ProtectCompose: func(context.Context, string) (orchestrator.ProtectReport, error) {
			return orchestrator.ProtectReport{
				Project: "demo",
				Created: 2,
			}, nil
		},
	}

	result, err := evaluator.EvaluateComposeOperation(context.Background(), "compose.yaml", OpComposeDown)
	if err != nil {
		t.Fatalf("evaluate compose operation: %v", err)
	}
	if !result.Allowed || !result.ProtectionRun {
		t.Fatalf("unexpected result: %#v", result)
	}
}
