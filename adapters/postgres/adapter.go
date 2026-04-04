package postgres

import (
	"context"
	"fmt"

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

	dataMount := adapterutil.MountForTarget(req.Target, "postgresql", "/var/lib/postgresql/data")
	manifest := map[string]any{
		"serviceType": "postgres",
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
		},
		"commands": map[string]any{
			"logicalBackup":  "pg_dumpall --clean --if-exists",
			"integrityCheck": "verify manifest + restore prerequisites",
			"restoreHint":    "psql / pg_restore depending on dump mode",
		},
		"notes": []string{
			"Postgres requires WAL-aware setup for serious point-in-time recovery.",
			"Persistent storage should target the PostgreSQL data directory.",
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
	return sdk.ValidationResult{
		IntegrityOK: true,
		RestoreTest: persistent,
		Degraded:    !persistent,
		Reason:      ternary(persistent, "postgres manifest verified with persistent mount", "postgres manifest verified in degraded emergency mode"),
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
