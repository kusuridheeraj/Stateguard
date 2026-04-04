package redis

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

func TestProtectIncludesRedisExecutionPlan(t *testing.T) {
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

	execution, _ := result.Manifest["execution"].(map[string]any)
	if execution == nil {
		t.Fatal("expected execution metadata in redis manifest")
	}
	capture, _ := execution["capture"].(map[string]any)
	if capture == nil {
		t.Fatal("expected capture execution plan")
	}
	if capture["captureCommand"] == "" {
		t.Fatal("expected capture command in redis manifest")
	}
	restore, _ := execution["restore"].(map[string]any)
	if restore == nil {
		t.Fatal("expected restore execution plan")
	}
	if restore["requiredArtifactID"] == "" {
		t.Fatal("expected required artifact id pattern")
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
    "execution": {
      "capture": {
        "supported": true,
        "captureCommand": "redis-cli --rdb /bundle/dump.rdb"
      },
      "restore": {
        "supported": true,
        "requiredArtifactID": "redis-<service>-<timestamp>",
        "expectedFiles": ["appendonly.aof", "dump.rdb"]
      }
    },
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

func TestValidateRedisRejectsWrongServiceType(t *testing.T) {
	adapter := New()
	temp := t.TempDir()
	path := filepath.Join(temp, "artifact.json")
	content := []byte(`{
  "manifest": {
    "serviceType": "postgres",
    "execution": {
      "restore": {
        "supported": true,
        "expectedFiles": ["dump.rdb"]
      }
    },
    "data": {
      "persistentMount": true,
      "persistenceMode": "durable-volume-backed"
    }
  }
}`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	if _, err := adapter.Validate(context.Background(), sdk.ArtifactRef{ID: "demo", Path: path}); err == nil {
		t.Fatal("expected validation failure for wrong service type")
	}
}

func TestRestoreRequiresStructuredRedisArtifactID(t *testing.T) {
	adapter := New()

	invalidID := fmt.Sprintf("invalid-%d", time.Now().UnixNano())
	if _, err := adapter.Restore(context.Background(), sdk.RestoreRequest{ArtifactID: invalidID}); err == nil {
		t.Fatal("expected invalid artifact id to fail restore")
	}

	validShapeButMissingBundle := fmt.Sprintf("redis-cache-%d", time.Now().UnixNano())
	if _, err := adapter.Restore(context.Background(), sdk.RestoreRequest{ArtifactID: validShapeButMissingBundle}); err == nil {
		t.Fatal("expected structured redis artifact id without persisted bundle to fail")
	}
}

func TestValidateRedisBundleWithExecutionMetadata(t *testing.T) {
	adapter := New()
	temp := t.TempDir()
	protected, err := adapter.Protect(context.Background(), sdk.ProtectRequest{
		Target: sdk.Target{
			Name:            "cache",
			Scope:           "demo",
			Runtime:         "compose",
			Image:           "redis:7",
			Mounts:          []string{"redis-data:/data"},
			PersistentMount: true,
		},
		ArtifactRoot:      filepath.Join(temp, "artifacts"),
		ValidationMode:    "hybrid",
		RestoreTestPolicy: "periodic",
	})
	if err != nil {
		t.Fatalf("protect: %v", err)
	}
	path := filepath.Join(temp, "artifact.json")
	payload := map[string]any{
		"manifest": map[string]any{
			"serviceType": "redis",
			"execution": map[string]any{
				"capture": map[string]any{
					"supported":      true,
					"captureCommand": "redis-cli --rdb /bundle/dump.rdb",
				},
				"restore": map[string]any{
					"supported":          true,
					"requiredArtifactID": "redis-<service>-<timestamp>",
					"expectedFiles":      []string{"appendonly.aof", "dump.rdb"},
				},
			},
			"data": map[string]any{
				"persistentMount": true,
				"persistenceMode": "durable-volume-backed",
			},
		},
	}
	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal artifact: %v", err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	result, err := adapter.Validate(context.Background(), sdk.ArtifactRef{ID: protected.Record.ID, Path: path})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if result.Degraded {
		t.Fatal("expected durable redis bundle to validate without degradation")
	}
	if !result.RestoreTest {
		t.Fatal("expected durable redis bundle to be restore-test eligible")
	}
}

func TestRestoreRedisBundleRequiresPersistedFiles(t *testing.T) {
	adapter := New()
	temp := t.TempDir()
	protected, err := adapter.Protect(context.Background(), sdk.ProtectRequest{
		Target: sdk.Target{
			Name:            "cache",
			Scope:           "demo",
			Runtime:         "compose",
			Image:           "redis:7",
			Mounts:          []string{"redis-data:/data"},
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
		t.Fatalf("expected redis restore success, got %#v", result)
	}
}
