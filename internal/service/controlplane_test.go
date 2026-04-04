package service

import (
	"context"
	"path/filepath"
	"testing"

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
