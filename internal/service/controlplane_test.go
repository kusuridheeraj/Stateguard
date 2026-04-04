package service

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kusuridheeraj/stateguard/internal/compose"
	"github.com/kusuridheeraj/stateguard/internal/config"
	"github.com/kusuridheeraj/stateguard/pkg/logging"
	"github.com/kusuridheeraj/stateguard/pkg/types"
)

func TestControlPlaneProtectComposeUpdatesStatus(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.Local.Path = filepath.Join(t.TempDir(), "artifacts")

	cp, err := NewControlPlane(logging.New(logging.Config{}), cfg, types.BuildInfo{Name: "stateguard"})
	if err != nil {
		t.Fatalf("new control plane: %v", err)
	}

	report, err := cp.ProtectCompose(context.Background(), filepath.Join("..", "..", "examples", "windows-wsl2-compose", "compose.yaml"))
	if err != nil {
		t.Fatalf("protect compose: %v", err)
	}
	if report.Created == 0 {
		t.Fatalf("expected artifacts to be created, got %#v", report)
	}

	status := cp.Status("test")
	if status.Artifacts.Count == 0 {
		t.Fatalf("expected artifact count in status, got %#v", status)
	}
	if status.ProtectedScopes == 0 {
		t.Fatalf("expected protected scopes in status, got %#v", status)
	}
}

func TestControlPlaneInterceptComposeDownExecutesRunner(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.Local.Path = filepath.Join(t.TempDir(), "artifacts")

	cp, err := NewControlPlane(logging.New(logging.Config{}), cfg, types.BuildInfo{Name: "stateguard"})
	if err != nil {
		t.Fatalf("new control plane: %v", err)
	}

	var command []string
	cp.composeRunner = compose.Runner{
		Exec: func(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
			command = append([]string{name}, args...)
			return []byte("down ok"), nil, nil
		},
	}

	path := filepath.Join("..", "..", "examples", "windows-wsl2-compose", "compose.yaml")
	result, err := cp.InterceptComposeDown(context.Background(), path, true, true)
	if err != nil {
		t.Fatalf("intercept compose down: %v", err)
	}
	if !result.Executed || result.RunResult == nil {
		t.Fatalf("expected execution result, got %#v", result)
	}
	if len(command) == 0 {
		t.Fatalf("expected docker command to be captured")
	}
}

func TestControlPlaneGuardKubeDeleteBlocksStatefulManifest(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.Local.Path = filepath.Join(t.TempDir(), "artifacts")

	cp, err := NewControlPlane(logging.New(logging.Config{}), cfg, types.BuildInfo{Name: "stateguard"})
	if err != nil {
		t.Fatalf("new control plane: %v", err)
	}

	result, err := cp.GuardKubeDelete(filepath.Join("..", "..", "examples", "kubernetes-beta", "manifests.yaml"))
	if err != nil {
		t.Fatalf("guard kube delete: %v", err)
	}
	if result.Allowed {
		t.Fatalf("expected stateful delete guard to block, got %#v", result)
	}
}

func TestControlPlaneRestoreArtifact(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.Local.Path = filepath.Join(t.TempDir(), "artifacts")

	cp, err := NewControlPlane(logging.New(logging.Config{}), cfg, types.BuildInfo{Name: "stateguard"})
	if err != nil {
		t.Fatalf("new control plane: %v", err)
	}

	report, err := cp.ProtectCompose(context.Background(), filepath.Join("..", "..", "examples", "windows-wsl2-compose", "compose.yaml"))
	if err != nil {
		t.Fatalf("protect compose: %v", err)
	}
	if len(report.Artifacts) == 0 {
		t.Fatal("expected at least one artifact")
	}

	restore, err := cp.RestoreArtifact(context.Background(), report.Artifacts[0].ID)
	if err != nil {
		t.Fatalf("restore artifact: %v", err)
	}
	if !restore.Recovered {
		t.Fatalf("expected restore success, got %#v", restore)
	}
}
