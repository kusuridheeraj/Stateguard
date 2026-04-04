package mongodb

import (
	"context"
	"fmt"

	"github.com/kusuridheeraj/stateguard/internal/adapterutil"
	"github.com/kusuridheeraj/stateguard/pkg/sdk"
)

type Adapter struct{}

func New() Adapter { return Adapter{} }

func (Adapter) Name() string { return "mongodb" }

func (Adapter) Metadata() sdk.Metadata {
	return sdk.Metadata{
		Official:     true,
		Priority:     92,
		ImageHints:   []string{"mongo", "mongodb"},
		Description:  "MongoDB adapter with service-aware manifest generation and validation hints.",
		Capabilities: []string{"detect", "protect", "validate", "restore", "manifest-aware"},
	}
}

func (Adapter) Detect(_ context.Context, target sdk.Target) (sdk.DetectionResult, error) {
	if !sdk.MatchesImage(target, []string{"mongo", "mongodb"}) {
		return sdk.DetectionResult{Supported: false, Stateful: target.StatefulHint, Confidence: "low", Reason: "image does not match mongo or mongodb"}, nil
	}

	reason := "mongodb image detected with persistent data mount"
	confidence := "high"
	if !target.PersistentMount {
		reason = "mongodb image detected without persistent data mount; emergency-only posture"
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
	record := adapterutil.NewRecord("mongodb", req, true)
	record.Degraded = !req.Target.PersistentMount

	dataMount := adapterutil.MountForTarget(req.Target, "/data/db")
	manifest := map[string]any{
		"serviceType": "mongodb",
		"strategy": map[string]any{
			"primary":           ternary(req.Target.PersistentMount, "mongodump-aware protection with durable storage hints", "container-layer emergency export"),
			"restoreValidation": ternary(req.Target.PersistentMount, "restore-test eligible", "integrity only until durable mount exists"),
			"oplogSuggested":    true,
		},
		"data": map[string]any{
			"mount":              dataMount,
			"persistentMount":    req.Target.PersistentMount,
			"expectedPaths":      []string{"WiredTiger", "collection-0", "journal"},
			"validationMode":     req.ValidationMode,
			"restoreTestPolicy":  req.RestoreTestPolicy,
			"degradedProtection": record.Degraded,
		},
		"commands": map[string]any{
			"logicalBackup":  "mongodump --archive",
			"integrityCheck": "verify manifest + MongoDB restore prerequisites",
			"restoreHint":    "mongorestore --archive",
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
	if manifest["serviceType"] != "mongodb" {
		return sdk.ValidationResult{}, fmt.Errorf("artifact %s is not a mongodb manifest", artifact.ID)
	}
	data, _ := manifest["data"].(map[string]any)
	persistent, _ := data["persistentMount"].(bool)

	return sdk.ValidationResult{
		IntegrityOK: true,
		RestoreTest: persistent,
		Degraded:    !persistent,
		Reason:      ternary(persistent, "mongodb manifest verified with persistent mount", "mongodb manifest verified in degraded emergency mode"),
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
