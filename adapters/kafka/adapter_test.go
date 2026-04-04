package kafka

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

func TestProtectIncludesKafkaSpecificManifest(t *testing.T) {
	adapter := New()
	result, err := adapter.Protect(context.Background(), sdk.ProtectRequest{
		Target: sdk.Target{
			Name:            "broker",
			Scope:           "demo",
			Runtime:         "compose",
			Image:           "kafka:latest",
			Mounts:          []string{"kafka-data:/var/lib/kafka"},
			PersistentMount: true,
		},
		ValidationMode:    "hybrid",
		RestoreTestPolicy: "periodic",
	})
	if err != nil {
		t.Fatalf("protect: %v", err)
	}

	if result.Manifest["serviceType"] != "kafka" {
		t.Fatalf("expected kafka service type, got %#v", result.Manifest["serviceType"])
	}
	execution, _ := result.Manifest["execution"].(map[string]any)
	if execution == nil {
		t.Fatal("expected kafka execution metadata")
	}
}

func TestValidateMarksKafkaDegradedWithoutPersistentMount(t *testing.T) {
	adapter := New()
	temp := t.TempDir()
	path := filepath.Join(temp, "artifact.json")
	content := []byte(`{
  "manifest": {
    "serviceType": "kafka",
    "execution": {
      "restore": {
        "supported": true
      }
    },
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

func TestRestoreRequiresPersistedKafkaBundle(t *testing.T) {
	adapter := New()
	invalidID := fmt.Sprintf("invalid-%d", time.Now().UnixNano())
	if _, err := adapter.Restore(context.Background(), sdk.RestoreRequest{ArtifactID: invalidID}); err == nil {
		t.Fatal("expected invalid kafka artifact id to fail")
	}

	temp := t.TempDir()
	protected, err := adapter.Protect(context.Background(), sdk.ProtectRequest{
		Target: sdk.Target{
			Name:            "broker",
			Scope:           "demo",
			Runtime:         "compose",
			Image:           "kafka:latest",
			Mounts:          []string{"kafka-data:/var/lib/kafka"},
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
		t.Fatalf("expected kafka restore success, got %#v", result)
	}
}
