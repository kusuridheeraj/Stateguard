package mysql

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kusuridheeraj/stateguard/internal/adapterutil"
	"github.com/kusuridheeraj/stateguard/pkg/sdk"
)

type Adapter struct{}

func New() Adapter { return Adapter{} }

func (Adapter) Name() string { return "mysql" }

func (Adapter) Metadata() sdk.Metadata {
	return sdk.Metadata{
		Official:     true,
		Priority:     93,
		ImageHints:   []string{"mysql", "mariadb"},
		Description:  "MySQL adapter with service-aware manifest generation and validation hints.",
		Capabilities: []string{"detect", "protect", "validate", "restore", "manifest-aware"},
	}
}

func (Adapter) Detect(_ context.Context, target sdk.Target) (sdk.DetectionResult, error) {
	if !sdk.MatchesImage(target, []string{"mysql", "mariadb"}) {
		return sdk.DetectionResult{Supported: false, Stateful: target.StatefulHint, Confidence: "low", Reason: "image does not match mysql or mariadb"}, nil
	}

	reason := "mysql-compatible image detected with persistent data mount"
	confidence := "high"
	if !target.PersistentMount {
		reason = "mysql-compatible image detected without persistent data mount; emergency-only posture"
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
	record := adapterutil.NewRecord("mysql", req, true)
	record.Degraded = !req.Target.PersistentMount

	dataMount := adapterutil.MountForTarget(req.Target, "/var/lib/mysql")
	dumpFileName := fmt.Sprintf("%s.sql", record.ID)
	manifest := map[string]any{
		"serviceType": "mysql",
		"artifact": map[string]any{
			"id":          record.ID,
			"bundleFiles": []string{"manifest.json", "checksum.sha256", "capture-plan.json", "restore.sh", "restore.ps1", "execution.json"},
		},
		"execution": map[string]any{
			"capture": map[string]any{
				"supported":      true,
				"mode":           ternary(req.Target.PersistentMount, "logical-dump", "degraded-emergency"),
				"captureCommand": "mysqldump --all-databases --single-transaction --quick",
				"artifactFile":   dumpFileName,
				"steps":          []string{"connect to mysql service", "run mysqldump", "seal checksum and manifest bundle"},
			},
			"restore": map[string]any{
				"supported":          true,
				"requiredArtifactID": "mysql-<service>-<timestamp>",
				"expectedFiles":      []string{"manifest.json", "checksum.sha256", "capture-plan.json", "restore.sh", "restore.ps1", "execution.json"},
				"restoreCommand":     fmt.Sprintf("mysql < %s", dumpFileName),
			},
		},
		"strategy": map[string]any{
			"primary":           ternary(req.Target.PersistentMount, "logical backup + binlog-aware protection", "container-layer emergency export"),
			"restoreValidation": ternary(req.Target.PersistentMount, "restore-test eligible", "integrity only until durable mount exists"),
			"binlogSuggested":   true,
		},
		"data": map[string]any{
			"mount":              dataMount,
			"persistentMount":    req.Target.PersistentMount,
			"expectedPaths":      []string{"ibdata1", "mysql", "performance_schema"},
			"validationMode":     req.ValidationMode,
			"restoreTestPolicy":  req.RestoreTestPolicy,
			"degradedProtection": record.Degraded,
			"dumpFileName":       dumpFileName,
		},
		"commands": map[string]any{
			"logicalBackup":  "mysqldump --all-databases --single-transaction --quick",
			"integrityCheck": "verify manifest + MySQL restore prerequisites",
			"restoreHint":    fmt.Sprintf("mysql < %s or physical restore depending on mode", dumpFileName),
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
	if manifest["serviceType"] != "mysql" {
		return sdk.ValidationResult{}, fmt.Errorf("artifact %s is not a mysql manifest", artifact.ID)
	}
	execution, _ := manifest["execution"].(map[string]any)
	restore, _ := execution["restore"].(map[string]any)
	if restore == nil {
		return sdk.ValidationResult{}, fmt.Errorf("artifact %s is missing mysql restore metadata", artifact.ID)
	}
	data, _ := manifest["data"].(map[string]any)
	persistent, _ := data["persistentMount"].(bool)
	if dumpFileName, _ := data["dumpFileName"].(string); dumpFileName == "" || !strings.HasSuffix(dumpFileName, ".sql") {
		return sdk.ValidationResult{}, fmt.Errorf("artifact %s is missing mysql dump metadata", artifact.ID)
	}

	return sdk.ValidationResult{
		IntegrityOK: true,
		RestoreTest: persistent,
		Degraded:    !persistent,
		Reason:      ternary(persistent, "mysql manifest verified with persistent mount", "mysql manifest verified in degraded emergency mode"),
	}, nil
}

func (Adapter) Restore(_ context.Context, req sdk.RestoreRequest) (sdk.RestoreResult, error) {
	if req.ArtifactID == "" || !strings.HasPrefix(req.ArtifactID, "mysql-") {
		return sdk.RestoreResult{}, fmt.Errorf("artifact id %q is not a mysql-generated bundle", req.ArtifactID)
	}
	if req.ArtifactPath == "" {
		return sdk.RestoreResult{}, fmt.Errorf("artifact id %q is missing manifest path", req.ArtifactID)
	}
	payload, err := adapterutil.ReadArtifactManifest(req.ArtifactPath)
	if err != nil {
		return sdk.RestoreResult{}, err
	}
	manifest, _ := payload["manifest"].(map[string]any)
	if serviceType, _ := manifest["serviceType"].(string); serviceType != "mysql" {
		return sdk.RestoreResult{}, fmt.Errorf("artifact id %q does not contain a mysql manifest", req.ArtifactID)
	}
	if req.BundleDir == "" {
		return sdk.RestoreResult{}, fmt.Errorf("artifact id %q is missing bundle path", req.ArtifactID)
	}
	for _, name := range []string{"manifest.json", "checksum.sha256", "capture-plan.json", "restore.sh", "restore.ps1", "execution.json"} {
		if _, err := os.Stat(filepath.Join(req.BundleDir, name)); err != nil {
			return sdk.RestoreResult{}, fmt.Errorf("artifact id %q is missing bundle file %s: %w", req.ArtifactID, name, err)
		}
	}
	return sdk.RestoreResult{
		Recovered: true,
		Details: map[string]any{
			"bundleDir": req.BundleDir,
			"mode":      "bundle-verified-mysql-restore",
		},
	}, nil
}

func ternary[T any](condition bool, whenTrue, whenFalse T) T {
	if condition {
		return whenTrue
	}
	return whenFalse
}
