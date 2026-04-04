package postgres

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kusuridheeraj/stateguard/internal/adapterutil"
	"github.com/kusuridheeraj/stateguard/pkg/sdk"
)

func TestProtectIncludesPostgresSpecificManifest(t *testing.T) {
	adapter := New()
	result, err := adapter.Protect(context.Background(), sdk.ProtectRequest{
		Target: sdk.Target{
			Name:            "db",
			Scope:           "demo",
			Runtime:         "compose",
			Image:           "postgres:16",
			Mounts:          []string{"db-data:/var/lib/postgresql/data"},
			PersistentMount: true,
		},
		ValidationMode:    "hybrid",
		RestoreTestPolicy: "periodic",
	})
	if err != nil {
		t.Fatalf("protect: %v", err)
	}

	if result.Manifest["serviceType"] != "postgres" {
		t.Fatalf("expected postgres service type, got %#v", result.Manifest["serviceType"])
	}
}

func TestValidateMarksDegradedWithoutPersistentMount(t *testing.T) {
	adapter := New()
	temp := t.TempDir()
	path := filepath.Join(temp, "artifact.json")
	payload := map[string]any{
		"manifest": map[string]any{
			"serviceType": "postgres",
			"data": map[string]any{
				"persistentMount": false,
			},
		},
	}
	if _, _, err := adapterutil.WriteManifestPreview(temp, "postgres", "db", payload); err != nil {
		t.Fatalf("write preview: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(temp, "previews", "postgres", "db.json"))
	if err != nil {
		t.Fatalf("read preview: %v", err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write artifact file: %v", err)
	}

	result, err := adapter.Validate(context.Background(), sdk.ArtifactRef{ID: "demo", Path: path})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !result.Degraded {
		t.Fatal("expected degraded validation result")
	}
}
