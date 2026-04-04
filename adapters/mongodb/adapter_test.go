package mongodb

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kusuridheeraj/stateguard/pkg/sdk"
)

func TestProtectIncludesMongoDBSpecificManifest(t *testing.T) {
	adapter := New()
	result, err := adapter.Protect(context.Background(), sdk.ProtectRequest{
		Target: sdk.Target{
			Name:            "mongo",
			Scope:           "demo",
			Runtime:         "compose",
			Image:           "mongo:7",
			Mounts:          []string{"mongo-data:/data/db"},
			PersistentMount: true,
		},
		ValidationMode:    "hybrid",
		RestoreTestPolicy: "periodic",
	})
	if err != nil {
		t.Fatalf("protect: %v", err)
	}

	if result.Manifest["serviceType"] != "mongodb" {
		t.Fatalf("expected mongodb service type, got %#v", result.Manifest["serviceType"])
	}
}

func TestValidateMarksMongoDBDegradedWithoutPersistentMount(t *testing.T) {
	adapter := New()
	temp := t.TempDir()
	path := filepath.Join(temp, "artifact.json")
	content := []byte(`{
  "manifest": {
    "serviceType": "mongodb",
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
