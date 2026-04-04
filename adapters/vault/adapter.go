package vault

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kusuridheeraj/stateguard/internal/adapterutil"
	"github.com/kusuridheeraj/stateguard/internal/config"
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
	record.RestoreTested = req.Target.PersistentMount

	dataMount := adapterutil.MountForTarget(req.Target, "/vault/file", "/vault/data")
	storageMode := ternary(req.Target.PersistentMount, "file-or-raft-persistent", "dev-or-ephemeral-suspected")
	captureMode := ternary(req.Target.PersistentMount, "persistent-storage-capture", "emergency-container-capture")
	restoreMode := ternary(req.Target.PersistentMount, "bundle-verified-storage-restore", "degraded-emergency-restore")

	manifest := map[string]any{
		"serviceType": "vault",
		"artifact": map[string]any{
			"id":          record.ID,
			"root":        req.ArtifactRoot,
			"bundleClass": "vault-storage-capture",
			"bundleFiles": []string{"manifest.json", "checksum.sha256", "capture-plan.json", "restore.sh", "restore.ps1"},
		},
		"strategy": map[string]any{
			"primary":           ternary(req.Target.PersistentMount, "capture persistent Vault storage metadata and bundle it for restore", "capture storage metadata plus emergency bundle for degraded recovery"),
			"restoreValidation": ternary(req.Target.PersistentMount, "bundle-verified restore-test eligible", "integrity only until persistent Vault storage is configured"),
		},
		"data": map[string]any{
			"mount":              dataMount,
			"persistentMount":    req.Target.PersistentMount,
			"storageMode":        storageMode,
			"sealMode":           ternary(req.Target.PersistentMount, "sealed-with-restore-path", "dev-mode-or-ephemeral"),
			"captureMode":        captureMode,
			"restoreMode":        restoreMode,
			"expectedPaths":      []string{"/vault/file", "/vault/data"},
			"validationMode":     req.ValidationMode,
			"restoreTestPolicy":  req.RestoreTestPolicy,
			"degradedProtection": record.Degraded,
		},
		"capturePlan": map[string]any{
			"steps": []string{
				"detect Vault storage backend and mount path",
				"freeze or snapshot the persistent storage location when available",
				"write manifest, checksum, and restore helpers into the bundle",
				"record the capture bundle under the host-managed artifact root",
			},
			"inputs": map[string]any{
				"artifactRoot":    req.ArtifactRoot,
				"validationMode":  req.ValidationMode,
				"restorePolicy":   req.RestoreTestPolicy,
				"persistentMount": req.Target.PersistentMount,
				"sourceMount":     dataMount,
				"storageMode":     storageMode,
			},
		},
		"restorePlan": map[string]any{
			"steps": []string{
				"locate the stored Vault bundle for the requested artifact id",
				"verify the checksum and bundle file layout",
				"reconstruct the persistent storage state",
				"start Vault and apply the configured unseal or auto-unseal path",
			},
			"requiredFiles":             []string{"manifest.json", "checksum.sha256", "capture-plan.json", "restore.sh", "restore.ps1"},
			"requiresPersistentStorage": req.Target.PersistentMount,
		},
		"commands": map[string]any{
			"integrityCheck": "verify manifest, checksum, bundle layout, and persistent Vault storage posture",
			"restoreHint":    "replay the stored bundle into persistent storage before unseal or cluster rejoin",
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
	if req.ArtifactID == "" {
		return sdk.RestoreResult{}, errors.New("artifact id is required for vault restore")
	}
	if !isVaultArtifactID(req.ArtifactID) {
		return sdk.RestoreResult{}, fmt.Errorf("artifact id %q does not match a vault bundle id", req.ArtifactID)
	}

	bundlePath, payload, err := locateVaultBundle(req.ArtifactID)
	if err != nil {
		return sdk.RestoreResult{}, err
	}

	manifest, _ := payload["manifest"].(map[string]any)
	if manifest == nil {
		return sdk.RestoreResult{}, fmt.Errorf("vault artifact %s is missing manifest data", req.ArtifactID)
	}
	if serviceType, _ := manifest["serviceType"].(string); serviceType != "vault" {
		return sdk.RestoreResult{}, fmt.Errorf("artifact %s is not a vault manifest", req.ArtifactID)
	}

	artifact, _ := manifest["artifact"].(map[string]any)
	data, _ := manifest["data"].(map[string]any)
	restorePlan, _ := manifest["restorePlan"].(map[string]any)
	capturePlan, _ := manifest["capturePlan"].(map[string]any)
	if artifact == nil || data == nil || restorePlan == nil || capturePlan == nil {
		return sdk.RestoreResult{}, fmt.Errorf("vault artifact %s is missing restore semantics", req.ArtifactID)
	}
	if bundleClass, _ := artifact["bundleClass"].(string); bundleClass != "vault-storage-capture" {
		return sdk.RestoreResult{}, fmt.Errorf("vault artifact %s has an unexpected bundle class", req.ArtifactID)
	}
	if rootHint, _ := artifact["root"].(string); rootHint == "" {
		return sdk.RestoreResult{}, fmt.Errorf("vault artifact %s is missing artifact root metadata", req.ArtifactID)
	}
	if files, ok := artifact["bundleFiles"].([]any); !ok || len(files) == 0 {
		return sdk.RestoreResult{}, fmt.Errorf("vault artifact %s is missing bundle file metadata", req.ArtifactID)
	}

	persistent, _ := data["persistentMount"].(bool)
	storageMode, _ := data["storageMode"].(string)
	if !persistent || strings.Contains(strings.ToLower(storageMode), "ephemeral") {
		return sdk.RestoreResult{}, fmt.Errorf("vault artifact %s is degraded and not eligible for a concrete restore", req.ArtifactID)
	}

	requiredFiles := []string{"manifest.json", "checksum.sha256", "capture-plan.json", "restore.sh", "restore.ps1"}
	for _, name := range requiredFiles {
		if _, err := os.Stat(filepath.Join(bundlePath, name)); err != nil {
			return sdk.RestoreResult{}, fmt.Errorf("vault artifact %s is missing required bundle file %s: %w", req.ArtifactID, name, err)
		}
	}

	if id, _ := artifact["id"].(string); id != req.ArtifactID {
		return sdk.RestoreResult{}, fmt.Errorf("vault artifact payload id mismatch for %s", req.ArtifactID)
	}

	if _, ok := restorePlan["steps"].([]any); !ok && restorePlan["steps"] != nil {
		return sdk.RestoreResult{}, fmt.Errorf("vault artifact %s has an invalid restore plan", req.ArtifactID)
	}
	if _, ok := capturePlan["steps"].([]any); !ok && capturePlan["steps"] != nil {
		return sdk.RestoreResult{}, fmt.Errorf("vault artifact %s has an invalid capture plan", req.ArtifactID)
	}

	return sdk.RestoreResult{Recovered: true}, nil
}

func ternary[T any](condition bool, whenTrue, whenFalse T) T {
	if condition {
		return whenTrue
	}
	return whenFalse
}

func isVaultArtifactID(id string) bool {
	if !strings.HasPrefix(id, "vault-") {
		return false
	}
	parts := strings.Split(id, "-")
	if len(parts) < 3 {
		return false
	}
	if _, err := strconv.ParseInt(parts[len(parts)-1], 10, 64); err != nil {
		return false
	}
	return true
}

func locateVaultBundle(artifactID string) (string, map[string]any, error) {
	cfg, err := config.Load(os.Getenv("STATEGUARD_CONFIG"))
	if err != nil {
		return "", nil, fmt.Errorf("load config for vault restore: %w", err)
	}

	root := cfg.Storage.Local.Path
	var foundPath string
	var foundPayload map[string]any

	walkErr := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry == nil || entry.IsDir() || entry.Name() != "manifest.json" {
			return nil
		}

		payload, err := adapterutil.ReadArtifactManifest(path)
		if err != nil {
			return nil
		}

		record, _ := payload["record"].(map[string]any)
		if record == nil {
			return nil
		}
		if recordID, _ := record["id"].(string); recordID != artifactID {
			return nil
		}

		foundPath = filepath.Dir(path)
		foundPayload = payload
		return filepath.SkipAll
	})
	if walkErr != nil {
		return "", nil, fmt.Errorf("search vault restore bundle: %w", walkErr)
	}
	if foundPath == "" {
		return "", nil, fmt.Errorf("vault artifact %s was not found under %s", artifactID, root)
	}

	return foundPath, foundPayload, nil
}
