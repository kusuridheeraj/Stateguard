package backupexec

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kusuridheeraj/stateguard/pkg/sdk"
)

func TestComposeExecutorWritesPlanWithoutLiveExecution(t *testing.T) {
	temp := t.TempDir()
	executor := NewComposeExecutor()

	result, err := executor.ExecuteProtection(context.Background(), "compose.yaml", sdk.Target{Name: "db"}, map[string]any{
		"serviceType": "postgres",
	}, temp)
	if err != nil {
		t.Fatalf("execute protection: %v", err)
	}
	if result.Mode != "planned" {
		t.Fatalf("expected planned mode, got %#v", result)
	}
	if _, err := os.Stat(filepath.Join(temp, "execution.json")); err != nil {
		t.Fatalf("expected execution.json: %v", err)
	}
}

func TestComposeExecutorRunsLiveCommands(t *testing.T) {
	temp := t.TempDir()
	var executed []string
	executor := ComposeExecutor{
		ExecuteLive: true,
		Runner: ExecFunc(func(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
			executed = append([]string{name}, args...)
			return []byte("ok"), nil, nil
		}),
	}

	result, err := executor.ExecuteProtection(context.Background(), "compose.yaml", sdk.Target{Name: "cache"}, map[string]any{
		"serviceType": "redis",
	}, temp)
	if err != nil {
		t.Fatalf("execute protection live: %v", err)
	}
	if result.Mode != "executed" {
		t.Fatalf("expected executed mode, got %#v", result)
	}
	if len(executed) == 0 || executed[0] != "docker" {
		t.Fatalf("expected docker command, got %#v", executed)
	}
	if _, err := os.Stat(filepath.Join(temp, "backup.redis.json")); err != nil {
		t.Fatalf("expected live output file: %v", err)
	}
}
