package service

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kusuridheeraj/stateguard/internal/backupexec"
	"github.com/kusuridheeraj/stateguard/internal/compose"
	"github.com/kusuridheeraj/stateguard/internal/config"
	"github.com/kusuridheeraj/stateguard/pkg/logging"
	"github.com/kusuridheeraj/stateguard/pkg/types"
)

type recordingCommandRunner struct {
	commands [][]string
}

func (r *recordingCommandRunner) Run(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
	command := append([]string{name}, args...)
	r.commands = append(r.commands, command)
	return []byte("ok"), nil, nil
}

func TestControlPlaneLiveProtectionRestoreAndInterceptFlow(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.Local.Path = filepath.Join(t.TempDir(), "artifacts")
	cfg.Runtime.Compose.LiveExecution = true

	cp, err := NewControlPlane(logging.New(logging.Config{}), cfg, types.BuildInfo{Name: "stateguard"})
	if err != nil {
		t.Fatalf("new control plane: %v", err)
	}

	backupRunner := &recordingCommandRunner{}
	cp.SetComposeBackupRunner(backupRunner)

	var composeCommand []string
	cp.SetComposeRunner(compose.Runner{
		Exec: func(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
			composeCommand = append([]string{name}, args...)
			return []byte("compose ok"), nil, nil
		},
	})

	composePath := filepath.Join("..", "..", "examples", "windows-wsl2-compose", "compose.yaml")
	report, err := cp.ProtectCompose(context.Background(), composePath)
	if err != nil {
		t.Fatalf("protect compose: %v", err)
	}
	if report.Created < 2 {
		t.Fatalf("expected multiple protected services, got %#v", report)
	}
	if len(backupRunner.commands) < report.Created {
		t.Fatalf("expected backup commands for protected services, got %d commands for %d artifacts", len(backupRunner.commands), report.Created)
	}

	restore, err := cp.RestoreArtifact(context.Background(), report.Artifacts[0].ID)
	if err != nil {
		t.Fatalf("restore artifact: %v", err)
	}
	if !restore.Recovered {
		t.Fatalf("expected restore success, got %#v", restore)
	}

	interceptResult, err := cp.InterceptDockerArgs(context.Background(), []string{"compose", "-f", composePath, "down", "-v"}, true)
	if err != nil {
		t.Fatalf("intercept docker args: %v", err)
	}
	if len(composeCommand) == 0 || composeCommand[0] != "docker" {
		t.Fatalf("expected intercepted compose command, got %#v", composeCommand)
	}

	resultPayload := interceptResult.Result
	if !resultPayload.Executed {
		t.Fatalf("expected executed interception result, got %#v", resultPayload)
	}
	if !strings.Contains(strings.Join(resultPayload.RunResult.Command, " "), "down") {
		t.Fatalf("expected compose down command, got %#v", resultPayload.RunResult.Command)
	}
}

func TestControlPlaneBlocksRawDockerPruneWithoutScopeMapping(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.Local.Path = filepath.Join(t.TempDir(), "artifacts")

	cp, err := NewControlPlane(logging.New(logging.Config{}), cfg, types.BuildInfo{Name: "stateguard"})
	if err != nil {
		t.Fatalf("new control plane: %v", err)
	}

	result, err := cp.InterceptDockerArgs(context.Background(), []string{"system", "prune", "-a"}, true)
	if err != nil {
		t.Fatalf("intercept docker prune: %v", err)
	}
	if result.Result.Allowed {
		t.Fatalf("expected raw docker prune to remain blocked, got %#v", result)
	}
	if result.Result.Scope != "host-global" {
		t.Fatalf("expected host-global scope, got %#v", result.Result.Scope)
	}
}

func TestControlPlaneKubeEnforcementCreatesProtectionArtifacts(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.Local.Path = filepath.Join(t.TempDir(), "artifacts")

	cp, err := NewControlPlane(logging.New(logging.Config{}), cfg, types.BuildInfo{Name: "stateguard"})
	if err != nil {
		t.Fatalf("new control plane: %v", err)
	}

	kubePath := filepath.Join("..", "..", "examples", "kubernetes-beta", "manifests.yaml")
	result, err := cp.EnforceKubeDelete(context.Background(), kubePath)
	if err != nil {
		t.Fatalf("enforce kube delete: %v", err)
	}

	protection, ok := result["protection"]
	if !ok {
		t.Fatalf("expected protection payload, got %#v", result)
	}
	if protection == nil {
		t.Fatalf("expected non-nil protection payload, got %#v", result)
	}
}

var _ backupexec.CommandRunner = (*recordingCommandRunner)(nil)
