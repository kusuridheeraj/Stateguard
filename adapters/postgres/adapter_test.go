package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kusuridheeraj/stateguard/internal/adapterutil"
	"github.com/kusuridheeraj/stateguard/pkg/sdk"
)

func TestProtectIncludesPostgresExecutionPlan(t *testing.T) {
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
		ArtifactRoot:      filepath.Join(t.TempDir(), "artifacts"),
		ValidationMode:    "hybrid",
		RestoreTestPolicy: "periodic",
	})
	if err != nil {
		t.Fatalf("protect: %v", err)
	}

	if result.Manifest["serviceType"] != "postgres" {
		t.Fatalf("expected postgres service type, got %#v", result.Manifest["serviceType"])
	}

	data, _ := result.Manifest["data"].(map[string]any)
	if dumpFileName, _ := data["dumpFileName"].(string); dumpFileName == "" || !strings.HasSuffix(dumpFileName, ".sql") {
		t.Fatalf("expected sql dump file name, got %#v", dumpFileName)
	}
	if dumpPathHint, _ := data["dumpFilePathHint"].(string); !strings.Contains(dumpPathHint, filepath.Join("demo", "db")) {
		t.Fatalf("expected dump path hint to include scope/service, got %#v", dumpPathHint)
	}

	commands, _ := result.Manifest["commands"].(map[string]any)
	if logicalBackup, _ := commands["logicalBackup"].(string); !strings.Contains(logicalBackup, "pg_dumpall") {
		t.Fatalf("expected pg_dumpall command, got %#v", logicalBackup)
	}

	execution, _ := result.Manifest["execution"].(map[string]any)
	steps, _ := execution["steps"].([]map[string]any)
	if len(steps) != 3 {
		t.Fatalf("expected 3 execution steps, got %#v", execution["steps"])
	}
}

func TestValidateAcceptsConcretePostgresArtifact(t *testing.T) {
	adapter := New()
	temp := t.TempDir()
	protected, err := adapter.Protect(context.Background(), sdk.ProtectRequest{
		Target: sdk.Target{
			Name:            "db",
			Scope:           "demo",
			Runtime:         "compose",
			Image:           "postgres:16",
			Mounts:          []string{"db-data:/var/lib/postgresql/data"},
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
	content, err := json.Marshal(map[string]any{"manifest": protected.Manifest})
	if err != nil {
		t.Fatalf("marshal artifact: %v", err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write artifact file: %v", err)
	}

	result, err := adapter.Validate(context.Background(), sdk.ArtifactRef{ID: protected.Record.ID, Path: path})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !result.RestoreTest || result.Degraded {
		t.Fatalf("expected non-degraded validation result, got %#v", result)
	}
}

func TestValidateRejectsMissingBackupMetadata(t *testing.T) {
	adapter := New()
	temp := t.TempDir()
	path := filepath.Join(temp, "artifact.json")
	content, err := json.Marshal(map[string]any{
		"manifest": map[string]any{
			"serviceType": "postgres",
			"data": map[string]any{
				"persistentMount": true,
			},
			"commands": map[string]any{
				"logicalBackup": "pg_dumpall --clean --if-exists",
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal artifact: %v", err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write artifact file: %v", err)
	}

	if _, err := adapter.Validate(context.Background(), sdk.ArtifactRef{ID: "demo", Path: path}); err == nil {
		t.Fatal("expected validation error for missing dump metadata")
	}
}

func TestRestoreRequiresConcreteProtectedArtifact(t *testing.T) {
	adapter := New()

	if result, err := adapter.Restore(context.Background(), sdk.RestoreRequest{ArtifactID: ""}); err == nil || result.Recovered {
		t.Fatalf("expected empty artifact id to fail, got result=%#v err=%v", result, err)
	}

	invalidID := fmt.Sprintf("invalid-%d", time.Now().UnixNano())
	if result, err := adapter.Restore(context.Background(), sdk.RestoreRequest{ArtifactID: invalidID}); err == nil || result.Recovered {
		t.Fatalf("expected missing cache entry to fail, got result=%#v err=%v", result, err)
	}
}

func TestRestoreSucceedsAfterProtect(t *testing.T) {
	adapter := New()
	temp := t.TempDir()
	protected, err := adapter.Protect(context.Background(), sdk.ProtectRequest{
		Target: sdk.Target{
			Name:            "db",
			Scope:           "demo",
			Runtime:         "compose",
			Image:           "postgres:16",
			Mounts:          []string{"db-data:/var/lib/postgresql/data"},
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
	payload, err := json.Marshal(map[string]any{"record": protected.Record, "manifest": protected.Manifest})
	if err != nil {
		t.Fatalf("marshal bundle: %v", err)
	}
	manifestPath := filepath.Join(bundleDir, "manifest.json")
	if err := os.WriteFile(manifestPath, payload, 0o600); err != nil {
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
		t.Fatal("expected restore to succeed for a previously protected artifact")
	}
	if result.Details["mode"] != "bundle-verified-postgres-restore" {
		t.Fatalf("unexpected restore details: %#v", result.Details)
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
				"dumpFileName":    "demo.sql",
			},
			"commands": map[string]any{
				"logicalBackup": "pg_dumpall --clean --if-exists",
			},
			"execution": map[string]any{
				"steps": []map[string]any{
					{"name": "resolve-target"},
					{"name": "capture-logical-backup"},
					{"name": "seal-backup-artifact"},
				},
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
