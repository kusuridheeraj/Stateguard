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

	result, err := evaluator.EvaluateComposeOperation(context.Background(), DockerArgsPlan{
		Operation:   OpComposeDown,
		ComposePath: "compose.yaml",
	})
	if err != nil {
		t.Fatalf("evaluate compose operation: %v", err)
	}
	if !result.Allowed || !result.ProtectionRun {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.Scope != "compose-project" {
		t.Fatalf("unexpected scope: %#v", result.Scope)
	}
}

func TestEvaluateDockerArgsBlocksRawVolumeRemove(t *testing.T) {
	evaluator := Evaluator{
		Mode: "fail-closed",
		ProtectCompose: func(context.Context, string) (orchestrator.ProtectReport, error) {
			t.Fatal("raw docker operations should not run compose protection")
			return orchestrator.ProtectReport{}, nil
		},
	}

	result, err := evaluator.EvaluateDockerArgs(context.Background(), DockerArgsPlan{
		Operation: OpDockerVolumeRemove,
		Targets:   []string{"cache-a"},
		Flags:     []string{"-f"},
	})
	if err != nil {
		t.Fatalf("evaluate docker args: %v", err)
	}
	if result.Allowed {
		t.Fatalf("expected raw volume remove to be blocked, got %#v", result)
	}
	if result.Scope != "host-global" {
		t.Fatalf("unexpected scope: %#v", result.Scope)
	}
}

func TestEvaluateDockerArgsBlocksSystemPruneWithWarnings(t *testing.T) {
	evaluator := Evaluator{Mode: "fail-closed"}

	result, err := evaluator.EvaluateDockerArgs(context.Background(), DockerArgsPlan{
		Operation: OpDockerSystemPrune,
		Flags:     []string{"--volumes", "-a"},
	})
	if err != nil {
		t.Fatalf("evaluate docker args: %v", err)
	}
	if result.Allowed {
		t.Fatalf("expected prune to be blocked, got %#v", result)
	}
	if len(result.Warnings) == 0 {
		t.Fatalf("expected warnings for prune, got %#v", result)
	}
}
