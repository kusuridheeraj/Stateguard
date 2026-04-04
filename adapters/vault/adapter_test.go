package vault

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
	if data["captureMode"] != "persistent-storage-capture" {
		t.Fatalf("unexpected capture mode: %#v", data["captureMode"])
	}
	if data["restoreMode"] != "bundle-verified-storage-restore" {
		t.Fatalf("unexpected restore mode: %#v", data["restoreMode"])
	}
	artifact, _ := result.Manifest["artifact"].(map[string]any)
	if artifact["bundleClass"] != "vault-storage-capture" {
		t.Fatalf("unexpected bundle class: %#v", artifact["bundleClass"])
	}
	if result.Record.RestoreTested != true {
		t.Fatalf("expected persistent Vault record to be restore-tested, got %#v", result.Record)
	}
	if _, ok := result.Manifest["capturePlan"].(map[string]any); !ok {
		t.Fatal("expected capturePlan metadata")
	}
	if _, ok := result.Manifest["restorePlan"].(map[string]any); !ok {
		t.Fatal("expected restorePlan metadata")
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
	if result.RestoreTest {
		t.Fatal("expected ephemeral Vault validation to skip restore test")
	}
}

func TestRestoreResolvesPersistedVaultBundle(t *testing.T) {
	adapter := New()
	temp := t.TempDir()
	artifactRoot := filepath.Join(temp, "artifacts")
	configPath := filepath.Join(temp, "safedata.yaml")
	configContent := []byte("version: \"1\"\nproject:\n  name: stateguard\n  environment: test\npolicy:\n  mode: fail-closed\n  validation:\n    strategy: hybrid\n    integrity_check: true\n    allow_degraded: false\n    restore_test:\n      enabled: true\n      cadence: periodic\n  retention:\n    window: 72h\n    max_disk_usage_percent: 20\nstorage:\n  local:\n    path: " + filepath.ToSlash(artifactRoot) + "\nruntime:\n  compose:\n    project_boundary: labels+compose_project\n  kubernetes:\n    mode: beta\ndaemon:\n  host: 127.0.0.1\n  port: 7010\napi:\n  host: 127.0.0.1\n  port: 7011\n")
	if err := os.WriteFile(configPath, configContent, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("STATEGUARD_CONFIG", configPath)

	protectResult, err := adapter.Protect(context.Background(), sdk.ProtectRequest{
		Target: sdk.Target{
			Name:            "vault",
			Scope:           "demo",
			Runtime:         "compose",
			Image:           "hashicorp/vault:latest",
			Mounts:          []string{"vault-data:/vault/file"},
			PersistentMount: true,
		},
		ArtifactRoot:      artifactRoot,
		ValidationMode:    "hybrid",
		RestoreTestPolicy: "periodic",
	})
	if err != nil {
		t.Fatalf("protect: %v", err)
	}

	bundleDir := filepath.Join(artifactRoot, "demo", "vault", protectResult.Record.ID)
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatalf("mkdir bundle: %v", err)
	}
	payload := map[string]any{
		"record":   protectResult.Record,
		"manifest": protectResult.Manifest,
	}
	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal bundle: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundleDir, "manifest.json"), content, 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	for _, name := range []string{"checksum.sha256", "capture-plan.json", "restore.sh", "restore.ps1"} {
		if err := os.WriteFile(filepath.Join(bundleDir, name), []byte("ok"), 0o600); err != nil {
			t.Fatalf("write bundle file %s: %v", name, err)
		}
	}

	result, err := adapter.Restore(context.Background(), sdk.RestoreRequest{ArtifactID: protectResult.Record.ID})
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if !result.Recovered {
		t.Fatalf("expected restore to recover bundle, got %#v", result)
	}
}

func TestRestoreRejectsMalformedVaultArtifactID(t *testing.T) {
	adapter := New()
	invalidID := fmt.Sprintf("invalid-%d", time.Now().UnixNano())
	if _, err := adapter.Restore(context.Background(), sdk.RestoreRequest{ArtifactID: invalidID}); err == nil {
		t.Fatal("expected malformed artifact id to fail")
	}
}
