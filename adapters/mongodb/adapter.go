package mongodb

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
	archiveName := fmt.Sprintf("%s.archive.gz", record.ID)
	manifest := map[string]any{
		"serviceType": "mongodb",
		"artifact": map[string]any{
			"id":          record.ID,
			"bundleFiles": []string{"manifest.json", "checksum.sha256", "capture-plan.json", "restore.sh", "restore.ps1", "execution.json"},
		},
		"execution": map[string]any{
			"capture": map[string]any{
				"supported":      true,
				"mode":           ternary(req.Target.PersistentMount, "archive-dump", "degraded-emergency"),
				"captureCommand": "mongodump --archive --gzip",
				"artifactFile":   archiveName,
				"steps":          []string{"connect to mongodb service", "run mongodump archive", "seal checksum and manifest bundle"},
			},
			"restore": map[string]any{
				"supported":          true,
				"requiredArtifactID": "mongodb-<service>-<timestamp>",
				"expectedFiles":      []string{"manifest.json", "checksum.sha256", "capture-plan.json", "restore.sh", "restore.ps1", "execution.json"},
				"restoreCommand":     fmt.Sprintf("mongorestore --archive=%s --gzip", archiveName),
			},
		},
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
			"archiveFileName":    archiveName,
		},
		"commands": map[string]any{
			"logicalBackup":  "mongodump --archive --gzip",
			"integrityCheck": "verify manifest + MongoDB restore prerequisites",
			"restoreHint":    fmt.Sprintf("mongorestore --archive=%s --gzip", archiveName),
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
	execution, _ := manifest["execution"].(map[string]any)
	restore, _ := execution["restore"].(map[string]any)
	if restore == nil {
		return sdk.ValidationResult{}, fmt.Errorf("artifact %s is missing mongodb restore metadata", artifact.ID)
	}
	data, _ := manifest["data"].(map[string]any)
	persistent, _ := data["persistentMount"].(bool)
	if archiveName, _ := data["archiveFileName"].(string); archiveName == "" || !strings.HasSuffix(archiveName, ".archive.gz") {
		return sdk.ValidationResult{}, fmt.Errorf("artifact %s is missing mongodb archive metadata", artifact.ID)
	}

	return sdk.ValidationResult{
		IntegrityOK: true,
		RestoreTest: persistent,
		Degraded:    !persistent,
		Reason:      ternary(persistent, "mongodb manifest verified with persistent mount", "mongodb manifest verified in degraded emergency mode"),
	}, nil
}

func (Adapter) Restore(_ context.Context, req sdk.RestoreRequest) (sdk.RestoreResult, error) {
	if req.ArtifactID == "" || !strings.HasPrefix(req.ArtifactID, "mongodb-") {
		return sdk.RestoreResult{}, fmt.Errorf("artifact id %q is not a mongodb-generated bundle", req.ArtifactID)
	}
	if req.ArtifactPath == "" {
		return sdk.RestoreResult{}, fmt.Errorf("artifact id %q is missing manifest path", req.ArtifactID)
	}
	payload, err := adapterutil.ReadArtifactManifest(req.ArtifactPath)
	if err != nil {
		return sdk.RestoreResult{}, err
	}
	manifest, _ := payload["manifest"].(map[string]any)
	if serviceType, _ := manifest["serviceType"].(string); serviceType != "mongodb" {
		return sdk.RestoreResult{}, fmt.Errorf("artifact id %q does not contain a mongodb manifest", req.ArtifactID)
	}
	if req.BundleDir == "" {
		return sdk.RestoreResult{}, fmt.Errorf("artifact id %q is missing bundle path", req.ArtifactID)
	}
	for _, name := range []string{"manifest.json", "checksum.sha256", "capture-plan.json", "restore.sh", "restore.ps1", "execution.json"} {
		if _, err := os.Stat(filepath.Join(req.BundleDir, name)); err != nil {
			return sdk.RestoreResult{}, fmt.Errorf("artifact id %q is missing bundle file %s: %w", req.ArtifactID, name, err)
		}
	}
	return sdk.RestoreResult{Recovered: true, Details: map[string]any{"bundleDir": req.BundleDir, "mode": "bundle-verified-mongodb-restore"}}, nil
}

func ternary[T any](condition bool, whenTrue, whenFalse T) T {
	if condition {
		return whenTrue
	}
	return whenFalse
}
