package vault

import (
	"context"
	"fmt"
	"strings"

	"github.com/kusuridheeraj/stateguard/internal/adapterutil"
	"github.com/kusuridheeraj/stateguard/pkg/sdk"
)

type Adapter struct{}

func New() Adapter { return Adapter{} }

func (Adapter) Name() string { return "vault" }

func (Adapter) Metadata() sdk.Metadata {
	return sdk.Metadata{
		Official:     true,
		Priority:     94,
		ImageHints:   []string{"vault", "hashicorp/vault"},
		Description:  "Vault adapter with storage-mode risk analysis and manifest-aware validation.",
		Capabilities: []string{"detect", "protect", "validate", "restore", "manifest-aware"},
	}
}

func (Adapter) Detect(_ context.Context, target sdk.Target) (sdk.DetectionResult, error) {
	if !sdk.MatchesImage(target, []string{"vault", "hashicorp/vault"}) {
		return sdk.DetectionResult{Supported: false, Stateful: target.StatefulHint, Confidence: "low", Reason: "image does not match vault"}, nil
	}

	reason := "vault image detected with persistent storage mount"
	confidence := "high"
	if !target.PersistentMount {
		reason = "vault image detected without persistent storage mount; dev/ephemeral posture assumed"
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
	record := adapterutil.NewRecord("vault", req, true)
	record.Degraded = !req.Target.PersistentMount

	dataMount := adapterutil.MountForTarget(req.Target, "/vault/file", "/vault/data")
	storageMode := ternary(req.Target.PersistentMount, "file-or-raft-persistent", "dev-or-ephemeral-suspected")

	manifest := map[string]any{
		"serviceType": "vault",
		"strategy": map[string]any{
			"primary":           ternary(req.Target.PersistentMount, "capture persistent Vault storage metadata", "emergency capture with suspected ephemeral storage"),
			"restoreValidation": ternary(req.Target.PersistentMount, "restore-test eligible", "integrity only until persistent Vault storage is configured"),
		},
		"data": map[string]any{
			"mount":              dataMount,
			"persistentMount":    req.Target.PersistentMount,
			"storageMode":        storageMode,
			"expectedPaths":      []string{"/vault/file", "/vault/data"},
			"validationMode":     req.ValidationMode,
			"restoreTestPolicy":  req.RestoreTestPolicy,
			"degradedProtection": record.Degraded,
		},
		"commands": map[string]any{
			"integrityCheck": "verify manifest + persistent Vault storage posture",
			"restoreHint":    "restore storage backend before unseal workflow",
			"operatorHint":   "Vault dev mode should not be treated as durable protection",
		},
		"notes": []string{
			"Vault requires persistent storage and seal management for serious recovery guarantees.",
			"Ephemeral or dev-like setups should be treated as degraded-risk.",
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
	if serviceType != "vault" {
		return sdk.ValidationResult{}, fmt.Errorf("artifact %s is not a vault manifest", artifact.ID)
	}

	data, _ := manifest["data"].(map[string]any)
	storageMode, _ := data["storageMode"].(string)
	persistent, _ := data["persistentMount"].(bool)
	degraded := !persistent || strings.Contains(storageMode, "ephemeral")

	return sdk.ValidationResult{
		IntegrityOK: true,
		RestoreTest: persistent,
		Degraded:    degraded,
		Reason:      ternary(degraded, "vault manifest verified with degraded storage posture", "vault manifest verified with persistent storage posture"),
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
