package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kusuridheeraj/stateguard/adapters/generic"
	"github.com/kusuridheeraj/stateguard/adapters/postgres"
	"github.com/kusuridheeraj/stateguard/internal/artifacts"
	"github.com/kusuridheeraj/stateguard/pkg/sdk"
)

func TestProtectComposeCreatesArtifacts(t *testing.T) {
	temp := t.TempDir()
	composePath := filepath.Join(temp, "compose.yaml")
	content := []byte("name: demo\nservices:\n  db:\n    image: postgres:16\n    volumes:\n      - db-data:/var/lib/postgresql/data\n  api:\n    image: demo-api:latest\nvolumes:\n  db-data: {}\n")
	if err := os.WriteFile(composePath, content, 0o600); err != nil {
		t.Fatalf("write compose file: %v", err)
	}

	store, err := artifacts.NewStore(filepath.Join(temp, "artifacts"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	registry := sdk.NewRegistry(postgres.New(), generic.New())
	report, err := NewProtector(store, registry).ProtectCompose(context.Background(), composePath)
	if err != nil {
		t.Fatalf("protect compose: %v", err)
	}

	if report.Created != 1 {
		t.Fatalf("expected 1 artifact, got %d", report.Created)
	}
	if len(store.List()) != 1 {
		t.Fatalf("expected artifact persisted in store")
	}
}
