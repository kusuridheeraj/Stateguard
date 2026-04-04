package kafka

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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
}

func TestValidateMarksKafkaDegradedWithoutPersistentMount(t *testing.T) {
	adapter := New()
	temp := t.TempDir()
	path := filepath.Join(temp, "artifact.json")
	content := []byte(`{
  "manifest": {
    "serviceType": "kafka",
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
