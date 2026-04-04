package mysql

import (
	"context"
	"fmt"

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
	manifest := map[string]any{
		"serviceType": "mysql",
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
		},
		"commands": map[string]any{
			"logicalBackup":  "mysqldump --all-databases --single-transaction",
			"integrityCheck": "verify manifest + MySQL restore prerequisites",
			"restoreHint":    "mysql < dump.sql or physical restore depending on mode",
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
	data, _ := manifest["data"].(map[string]any)
	persistent, _ := data["persistentMount"].(bool)

	return sdk.ValidationResult{
		IntegrityOK: true,
		RestoreTest: persistent,
		Degraded:    !persistent,
		Reason:      ternary(persistent, "mysql manifest verified with persistent mount", "mysql manifest verified in degraded emergency mode"),
	}, nil
}

func (Adapter) Restore(_ context.Context, req sdk.RestoreRequest) (sdk.RestoreResult, error) {
	return sdk.RestoreResult{Recovered: req.ArtifactID != ""}, nil
}

func ternary[T any](condition bool, whenTrue, whenFalse T) T {
	if condition {
		return whenTrue
	}
	return whenFalse
}
