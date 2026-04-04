package mysql

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kusuridheeraj/stateguard/pkg/sdk"
)

func TestProtectIncludesMySQLSpecificManifest(t *testing.T) {
	adapter := New()
	result, err := adapter.Protect(context.Background(), sdk.ProtectRequest{
		Target: sdk.Target{
			Name:            "mysql",
			Scope:           "demo",
			Runtime:         "compose",
			Image:           "mysql:8",
			Mounts:          []string{"mysql-data:/var/lib/mysql"},
			PersistentMount: true,
		},
		ValidationMode:    "hybrid",
		RestoreTestPolicy: "periodic",
	})
	if err != nil {
		t.Fatalf("protect: %v", err)
	}

	if result.Manifest["serviceType"] != "mysql" {
		t.Fatalf("expected mysql service type, got %#v", result.Manifest["serviceType"])
	}
}

func TestValidateMarksMySQLDegradedWithoutPersistentMount(t *testing.T) {
	adapter := New()
	temp := t.TempDir()
	path := filepath.Join(temp, "artifact.json")
	content := []byte(`{
  "manifest": {
    "serviceType": "mysql",
    "data": {
      "persistentMount": false
    }
  }
}`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	result, err := adapter.Validate(context.Background(), sdk.ArtifactRef{ID: "demo", Path: path})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !result.Degraded {
		t.Fatal("expected degraded validation result")
	}
}
