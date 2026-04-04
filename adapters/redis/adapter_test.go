package redis

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kusuridheeraj/stateguard/pkg/sdk"
)

func TestProtectIncludesRedisPersistenceMode(t *testing.T) {
	adapter := New()
	result, err := adapter.Protect(context.Background(), sdk.ProtectRequest{
		Target: sdk.Target{
			Name:            "cache",
			Scope:           "demo",
			Runtime:         "compose",
			Image:           "redis:7",
			Mounts:          []string{"redis-data:/data"},
			PersistentMount: true,
		},
		ValidationMode:    "hybrid",
		RestoreTestPolicy: "periodic",
	})
	if err != nil {
		t.Fatalf("protect: %v", err)
	}

	data, _ := result.Manifest["data"].(map[string]any)
	if data["persistenceMode"] != "durable-volume-backed" {
		t.Fatalf("unexpected persistence mode: %#v", data["persistenceMode"])
	}
}

func TestValidateRedisDegradedForMemoryMode(t *testing.T) {
	adapter := New()
	temp := t.TempDir()
	path := filepath.Join(temp, "artifact.json")
	content := []byte(`{
  "manifest": {
    "serviceType": "redis",
    "data": {
      "persistentMount": false,
      "persistenceMode": "memory-or-container-layer-only"
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
