package orchestrator

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kusuridheeraj/stateguard/adapters/generic"
	"github.com/kusuridheeraj/stateguard/adapters/kafka"
	"github.com/kusuridheeraj/stateguard/adapters/mongodb"
	"github.com/kusuridheeraj/stateguard/adapters/mysql"
	"github.com/kusuridheeraj/stateguard/adapters/postgres"
	"github.com/kusuridheeraj/stateguard/adapters/redis"
	"github.com/kusuridheeraj/stateguard/adapters/vault"
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
	if report.Artifacts[0].BundleDir == "" || report.Artifacts[0].ChecksumSHA256 == "" {
		t.Fatalf("expected bundle metadata, got %#v", report.Artifacts[0])
	}
}

func TestProtectComposeCreatesAllOfficialAdapterArtifacts(t *testing.T) {
	temp := t.TempDir()
	composePath := filepath.Join(temp, "compose.yaml")
	content := []byte("name: demo\nservices:\n  db:\n    image: postgres:16\n    volumes:\n      - db-data:/var/lib/postgresql/data\n  cache:\n    image: redis:7\n    volumes:\n      - redis-data:/data\n  secrets:\n    image: hashicorp/vault:latest\n    volumes:\n      - vault-data:/vault/file\n  mysql:\n    image: mysql:8\n    volumes:\n      - mysql-data:/var/lib/mysql\n  mongo:\n    image: mongo:7\n    volumes:\n      - mongo-data:/data/db\n  broker:\n    image: kafka:latest\n    volumes:\n      - kafka-data:/var/lib/kafka\nvolumes:\n  db-data: {}\n  redis-data: {}\n  vault-data: {}\n  mysql-data: {}\n  mongo-data: {}\n  kafka-data: {}\n")
	if err := os.WriteFile(composePath, content, 0o600); err != nil {
		t.Fatalf("write compose file: %v", err)
	}

	store, err := artifacts.NewStore(filepath.Join(temp, "artifacts"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	registry := sdk.NewRegistry(
		postgres.New(),
		redis.New(),
		vault.New(),
		mysql.New(),
		mongodb.New(),
		kafka.New(),
		generic.New(),
	)
	report, err := NewProtector(store, registry).ProtectCompose(context.Background(), composePath)
	if err != nil {
		t.Fatalf("protect compose: %v", err)
	}
	if report.Created != 6 {
		t.Fatalf("expected 6 artifacts, got %d", report.Created)
	}

	expectations := map[string]string{
		"db":      "postgres",
		"cache":   "redis",
		"secrets": "vault",
		"mysql":   "mysql",
		"mongo":   "mongodb",
		"broker":  "kafka",
	}
	for _, artifact := range report.Artifacts {
		content, err := os.ReadFile(artifact.Path)
		if err != nil {
			t.Fatalf("read artifact file: %v", err)
		}
		var payload map[string]any
		if err := json.Unmarshal(content, &payload); err != nil {
			t.Fatalf("decode artifact file: %v", err)
		}
		manifest, _ := payload["manifest"].(map[string]any)
		if manifest["serviceType"] != expectations[artifact.Service] {
			t.Fatalf("unexpected manifest service type for %s: %#v", artifact.Service, manifest["serviceType"])
		}
		for _, relative := range []string{"checksum.sha256", "capture-plan.json", "restore.sh", "restore.ps1"} {
			if _, err := os.Stat(filepath.Join(artifact.BundleDir, relative)); err != nil {
				t.Fatalf("expected bundle file %s for %s: %v", relative, artifact.Service, err)
			}
		}
		if _, err := os.Stat(filepath.Join(artifact.BundleDir, "execution.json")); err != nil {
			t.Fatalf("expected execution metadata for %s: %v", artifact.Service, err)
		}
	}
}

func TestProtectComposeCreatesServiceSpecificArtifacts(t *testing.T) {
	temp := t.TempDir()
	composePath := filepath.Join(temp, "compose.yaml")
	content := []byte("name: demo\nservices:\n  db:\n    image: postgres:16\n    volumes:\n      - db-data:/var/lib/postgresql/data\n  cache:\n    image: redis:7\n    volumes:\n      - redis-data:/data\n  secrets:\n    image: hashicorp/vault:latest\n    volumes:\n      - vault-data:/vault/file\nvolumes:\n  db-data: {}\n  redis-data: {}\n  vault-data: {}\n")
	if err := os.WriteFile(composePath, content, 0o600); err != nil {
		t.Fatalf("write compose file: %v", err)
	}

	store, err := artifacts.NewStore(filepath.Join(temp, "artifacts"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	registry := sdk.NewRegistry(postgres.New(), redis.New(), vault.New(), generic.New())
	report, err := NewProtector(store, registry).ProtectCompose(context.Background(), composePath)
	if err != nil {
		t.Fatalf("protect compose: %v", err)
	}
	if report.Created != 3 {
		t.Fatalf("expected 3 artifacts, got %d", report.Created)
	}

	expectations := map[string]string{
		"db":      "postgres",
		"cache":   "redis",
		"secrets": "vault",
	}
	for _, artifact := range report.Artifacts {
		content, err := os.ReadFile(artifact.Path)
		if err != nil {
			t.Fatalf("read artifact file: %v", err)
		}
		var payload map[string]any
		if err := json.Unmarshal(content, &payload); err != nil {
			t.Fatalf("decode artifact file: %v", err)
		}
		manifest, _ := payload["manifest"].(map[string]any)
		if manifest["serviceType"] != expectations[artifact.Service] {
			t.Fatalf("unexpected manifest service type for %s: %#v", artifact.Service, manifest["serviceType"])
		}
		if artifact.BundleDir == "" {
			t.Fatalf("expected bundle dir for %s", artifact.Service)
		}
	}
}
