package compose

import (
	"context"
	"testing"
)

func TestRunnerDownBuildsDockerCommand(t *testing.T) {
	var command []string
	runner := Runner{
		Exec: func(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
			command = append([]string{name}, args...)
			return []byte("ok"), nil, nil
		},
	}

	result, err := runner.Down(context.Background(), "compose.yaml", true, true)
	if err != nil {
		t.Fatalf("runner down: %v", err)
	}
	if len(command) == 0 || command[0] != "docker" {
		t.Fatalf("unexpected command: %#v", command)
	}
	foundVolumes := false
	for _, arg := range command {
		if arg == "-v" {
			foundVolumes = true
			break
		}
	}
	if !foundVolumes {
		t.Fatalf("expected -v in command, got %#v", command)
	}
	if result.Stdout != "ok" {
		t.Fatalf("unexpected stdout: %#v", result.Stdout)
	}
}
