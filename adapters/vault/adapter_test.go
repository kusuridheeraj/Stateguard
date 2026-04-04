package vault

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kusuridheeraj/stateguard/pkg/sdk"
)

func TestProtectIncludesVaultStorageMode(t *testing.T) {
	adapter := New()
	result, err := adapter.Protect(context.Background(), sdk.ProtectRequest{
		Target: sdk.Target{
			Name:            "vault",
			Scope:           "demo",
			Runtime:         "compose",
			Image:           "hashicorp/vault:latest",
			Mounts:          []string{"vault-data:/vault/file"},
			PersistentMount: true,
		},
		ValidationMode:    "hybrid",
		RestoreTestPolicy: "periodic",
	})
	if err != nil {
		t.Fatalf("protect: %v", err)
	}

	data, _ := result.Manifest["data"].(map[string]any)
	if data["storageMode"] != "file-or-raft-persistent" {
		t.Fatalf("unexpected storage mode: %#v", data["storageMode"])
	}
}

func TestValidateVaultDegradedForEphemeralMode(t *testing.T) {
	adapter := New()
	temp := t.TempDir()
	path := filepath.Join(temp, "artifact.json")
	content := []byte(`{
  "manifest": {
    "serviceType": "vault",
    "data": {
      "persistentMount": false,
      "storageMode": "dev-or-ephemeral-suspected"
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
