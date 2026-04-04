package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kusuridheeraj/stateguard/internal/adapterutil"
	"github.com/kusuridheeraj/stateguard/pkg/sdk"
)

type Adapter struct{}

func New() Adapter { return Adapter{} }

func (Adapter) Name() string { return "postgres" }

func (Adapter) Metadata() sdk.Metadata {
	return sdk.Metadata{
		Official:     true,
		Priority:     100,
		ImageHints:   []string{"postgres"},
		Description:  "Postgres adapter with service-aware manifest generation and validation hints.",
		Capabilities: []string{"detect", "protect", "validate", "restore", "manifest-aware"},
	}
}

func (Adapter) Detect(_ context.Context, target sdk.Target) (sdk.DetectionResult, error) {
	if !sdk.MatchesImage(target, []string{"postgres"}) {
		return sdk.DetectionResult{Supported: false, Stateful: target.StatefulHint, Confidence: "low", Reason: "image does not match postgres"}, nil
	}

	reason := "postgres image detected with persistent data mount"
	confidence := "high"
	if !target.PersistentMount {
		reason = "postgres image detected without persistent data mount; emergency-only posture"
		confidence = "medium"
	}

	return sdk.DetectionResult{
		Supported:  true,
		Stateful:   true,
		Confidence: confidence,
		Reason:     reason,
	}, nil
}

func (Adapter) Protect(_ context.Context, req sdk.ProtectRequest) (sdk.ProtectResult, error) {
	record := adapterutil.NewRecord("postgres", req, true)
	record.Degraded = !req.Target.PersistentMount
	dumpFileName := fmt.Sprintf("%s.sql", record.ID)
	artifactRootHint := req.ArtifactRoot
	if artifactRootHint == "" {
		artifactRootHint = "<artifact-root>"
	}
	bundleHint := filepath.Join(artifactRootHint, sanitizeComponent(record.Scope), sanitizeComponent(record.Service), record.ID)
	dumpFilePath := filepath.Join(bundleHint, dumpFileName)
	executionSteps := []map[string]any{
		{
			"name":    "resolve-target",
			"command": "docker compose exec -T postgres sh -lc 'test -d /var/lib/postgresql/data'",
		},
		{
			"name":    "capture-logical-backup",
			"command": fmt.Sprintf("pg_dumpall --clean --if-exists --file %s", dumpFilePath),
		},
		{
			"name":    "seal-backup-artifact",
			"command": fmt.Sprintf("sha256sum %s > %s.sha256", dumpFilePath, dumpFilePath),
		},
	}

	dataMount := adapterutil.MountForTarget(req.Target, "postgresql", "/var/lib/postgresql/data")
	manifest := map[string]any{
		"serviceType": "postgres",
		"artifact": map[string]any{
			"id":          record.ID,
			"bundleFiles": []string{"manifest.json", "checksum.sha256", "capture-plan.json", "restore.sh", "restore.ps1", "execution.json"},
		},
		"strategy": map[string]any{
			"primary":               ternary(req.Target.PersistentMount, "volume-aware logical backup + wal coordination", "container-layer emergency export"),
			"restoreValidation":     ternary(req.Target.PersistentMount, "restore-test eligible", "integrity only until durable mount exists"),
			"walArchivingSuggested": true,
		},
		"data": map[string]any{
			"mount":              dataMount,
			"persistentMount":    req.Target.PersistentMount,
			"expectedPaths":      []string{"PG_VERSION", "base", "global", "pg_wal"},
			"validationMode":     req.ValidationMode,
			"restoreTestPolicy":  req.RestoreTestPolicy,
			"degradedProtection": record.Degraded,
			"dumpFileName":       dumpFileName,
			"dumpFilePathHint":   dumpFilePath,
			"artifactRootHint":   artifactRootHint,
		},
		"commands": map[string]any{
			"logicalBackup":     fmt.Sprintf("pg_dumpall --clean --if-exists --file %s", dumpFilePath),
			"integrityCheck":    "verify manifest + restore prerequisites + checksum",
			"restoreHint":       fmt.Sprintf("psql < %s or pg_restore depending on dump mode", dumpFilePath),
			"restoreValidation": "compare checksum and replay readiness before restore",
		},
		"execution": map[string]any{
			"mode":  ternary(req.Target.PersistentMount, "durable", "degraded"),
			"steps": executionSteps,
		},
		"notes": []string{
			"Postgres requires WAL-aware setup for serious point-in-time recovery.",
			"Persistent storage should target the PostgreSQL data directory.",
			"Logical backup metadata records the intended dump filename and execution steps even when live execution is deferred.",
		},
	}

	return sdk.ProtectResult{Record: record, Manifest: manifest}, nil
}

func (Adapter) Validate(_ context.Context, artifact sdk.ArtifactRef) (sdk.ValidationResult, error) {
	payload, err := adapterutil.ReadArtifactManifest(artifact.Path)
	if err != nil {
		return sdk.ValidationResult{}, err
	}

	manifest, _ := payload["manifest"].(map[string]any)
	serviceType, _ := manifest["serviceType"].(string)
	if serviceType != "postgres" {
		return sdk.ValidationResult{}, fmt.Errorf("artifact %s is not a postgres manifest", artifact.ID)
	}

	data, _ := manifest["data"].(map[string]any)
	persistent, _ := data["persistentMount"].(bool)
	dumpFileName, _ := data["dumpFileName"].(string)
	commands, _ := manifest["commands"].(map[string]any)
	logicalBackup, _ := commands["logicalBackup"].(string)
	execution, _ := manifest["execution"].(map[string]any)
	steps, _ := execution["steps"].([]any)
	if dumpFileName == "" || !strings.HasSuffix(dumpFileName, ".sql") {
		return sdk.ValidationResult{}, fmt.Errorf("artifact %s is missing postgres dump metadata", artifact.ID)
	}
	if !strings.Contains(logicalBackup, "pg_dumpall") {
		return sdk.ValidationResult{}, fmt.Errorf("artifact %s is missing postgres backup command metadata", artifact.ID)
	}
	if len(steps) < 3 {
		return sdk.ValidationResult{}, fmt.Errorf("artifact %s does not contain enough postgres execution steps", artifact.ID)
	}
	return sdk.ValidationResult{
		IntegrityOK: true,
		RestoreTest: persistent,
		Degraded:    !persistent,
		Reason:      ternary(persistent, "postgres manifest verified with persistent mount", "postgres manifest verified in degraded emergency mode"),
	}, nil
}

func (Adapter) Restore(_ context.Context, req sdk.RestoreRequest) (sdk.RestoreResult, error) {
	if req.ArtifactID == "" {
		return sdk.RestoreResult{}, errors.New("artifact id is required")
	}
	if !strings.HasPrefix(req.ArtifactID, "postgres-") {
		return sdk.RestoreResult{}, fmt.Errorf("artifact %s is not a postgres backup", req.ArtifactID)
	}
	if req.ArtifactPath == "" {
		return sdk.RestoreResult{}, fmt.Errorf("artifact %s is missing persisted manifest path", req.ArtifactID)
	}

	payload, err := adapterutil.ReadArtifactManifest(req.ArtifactPath)
	if err != nil {
		return sdk.RestoreResult{}, err
	}
	manifest, _ := payload["manifest"].(map[string]any)
	if serviceType, _ := manifest["serviceType"].(string); serviceType != "postgres" {
		return sdk.RestoreResult{}, fmt.Errorf("artifact %s does not reference postgres", req.ArtifactID)
	}
	commands, _ := manifest["commands"].(map[string]any)
	if logicalBackup, _ := commands["logicalBackup"].(string); !strings.Contains(logicalBackup, "pg_dumpall") {
		return sdk.RestoreResult{}, fmt.Errorf("artifact %s is missing restore-ready backup metadata", req.ArtifactID)
	}
	data, _ := manifest["data"].(map[string]any)
	if dumpFileName, _ := data["dumpFileName"].(string); dumpFileName == "" || !strings.HasSuffix(dumpFileName, ".sql") {
		return sdk.RestoreResult{}, fmt.Errorf("artifact %s is missing dump filename metadata", req.ArtifactID)
	}
	execution, _ := manifest["execution"].(map[string]any)
	if executionStepCount(execution["steps"]) < 3 {
		return sdk.RestoreResult{}, fmt.Errorf("artifact %s is missing restore execution metadata", req.ArtifactID)
	}
	if req.BundleDir == "" {
		return sdk.RestoreResult{}, fmt.Errorf("artifact %s is missing persisted bundle path", req.ArtifactID)
	}
	for _, name := range []string{"manifest.json", "checksum.sha256", "capture-plan.json", "restore.sh", "restore.ps1", "execution.json"} {
		if _, err := os.Stat(filepath.Join(req.BundleDir, name)); err != nil {
			return sdk.RestoreResult{}, fmt.Errorf("artifact %s is missing bundle file %s: %w", req.ArtifactID, name, err)
		}
	}
	return sdk.RestoreResult{
		Recovered: true,
		Details: map[string]any{
			"bundleDir": req.BundleDir,
			"mode":      "bundle-verified-postgres-restore",
		},
	}, nil
}

func ternary[T any](condition bool, whenTrue, whenFalse T) T {
	if condition {
		return whenTrue
	}
	return whenFalse
}

func sanitizeComponent(value string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-")
	return replacer.Replace(value)
}

func executionStepCount(value any) int {
	switch steps := value.(type) {
	case []any:
		return len(steps)
	case []map[string]any:
		return len(steps)
	default:
		return 0
	}
}
