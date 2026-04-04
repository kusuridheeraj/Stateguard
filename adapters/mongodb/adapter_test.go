package mongodb

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	data, _ := result.Manifest["data"].(map[string]any)
	if archiveName, _ := data["archiveFileName"].(string); archiveName == "" {
		t.Fatal("expected mongodb archive metadata")
	}
}

func TestValidateMarksMongoDBDegradedWithoutPersistentMount(t *testing.T) {
	adapter := New()
	temp := t.TempDir()
	path := filepath.Join(temp, "artifact.json")
	content := []byte(`{
  "manifest": {
    "serviceType": "mongodb",
    "execution": {
      "restore": {
        "supported": true
      }
    },
    "data": {
      "persistentMount": false,
      "archiveFileName": "backup.archive.gz"
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

func TestRestoreRequiresPersistedMongoBundle(t *testing.T) {
	adapter := New()
	invalidID := fmt.Sprintf("invalid-%d", time.Now().UnixNano())
	if _, err := adapter.Restore(context.Background(), sdk.RestoreRequest{ArtifactID: invalidID}); err == nil {
		t.Fatal("expected invalid mongodb artifact id to fail")
	}

	temp := t.TempDir()
	protected, err := adapter.Protect(context.Background(), sdk.ProtectRequest{
		Target: sdk.Target{
			Name:            "mongo",
			Scope:           "demo",
			Runtime:         "compose",
			Image:           "mongo:7",
			Mounts:          []string{"mongo-data:/data/db"},
			PersistentMount: true,
		},
		ArtifactRoot:      filepath.Join(temp, "artifacts"),
		ValidationMode:    "hybrid",
		RestoreTestPolicy: "periodic",
	})
	if err != nil {
		t.Fatalf("protect: %v", err)
	}
	bundleDir := filepath.Join(temp, "bundle")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatalf("mkdir bundle: %v", err)
	}
	content, err := json.Marshal(map[string]any{"record": protected.Record, "manifest": protected.Manifest})
	if err != nil {
		t.Fatalf("marshal bundle: %v", err)
	}
	manifestPath := filepath.Join(bundleDir, "manifest.json")
	if err := os.WriteFile(manifestPath, content, 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	for _, name := range []string{"checksum.sha256", "capture-plan.json", "restore.sh", "restore.ps1", "execution.json"} {
		if err := os.WriteFile(filepath.Join(bundleDir, name), []byte("ok"), 0o600); err != nil {
			t.Fatalf("write bundle file %s: %v", name, err)
		}
	}
	result, err := adapter.Restore(context.Background(), sdk.RestoreRequest{
		ArtifactID:   protected.Record.ID,
		ArtifactPath: manifestPath,
		BundleDir:    bundleDir,
	})
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if !result.Recovered {
		t.Fatalf("expected mongodb restore success, got %#v", result)
	}
}
