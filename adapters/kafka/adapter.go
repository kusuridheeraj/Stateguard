package kafka

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

func (Adapter) Name() string { return "kafka" }

func (Adapter) Metadata() sdk.Metadata {
	return sdk.Metadata{
		Official:     true,
		Priority:     91,
		ImageHints:   []string{"kafka", "redpanda"},
		Description:  "Kafka adapter with log-dir-aware manifest generation and validation hints.",
		Capabilities: []string{"detect", "protect", "validate", "restore", "manifest-aware"},
	}
}

func (Adapter) Detect(_ context.Context, target sdk.Target) (sdk.DetectionResult, error) {
	if !sdk.MatchesImage(target, []string{"kafka", "redpanda"}) {
		return sdk.DetectionResult{Supported: false, Stateful: target.StatefulHint, Confidence: "low", Reason: "image does not match kafka or redpanda"}, nil
	}

	reason := "kafka-compatible broker detected with persistent log mount"
	confidence := "high"
	if !target.PersistentMount {
		reason = "kafka-compatible broker detected without persistent log mount; queue durability is degraded"
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
	record := adapterutil.NewRecord("kafka", req, true)
	record.Degraded = !req.Target.PersistentMount

	dataMount := adapterutil.MountForTarget(req.Target, "/var/lib/kafka", "/bitnami/kafka", "/var/lib/redpanda")
	brokerType := "kafka"
	if strings.Contains(strings.ToLower(req.Target.Image), "redpanda") {
		brokerType = "redpanda"
	}

	manifest := map[string]any{
		"serviceType": "kafka",
		"artifact": map[string]any{
			"id":          record.ID,
			"bundleFiles": []string{"manifest.json", "checksum.sha256", "capture-plan.json", "restore.sh", "restore.ps1", "execution.json"},
		},
		"execution": map[string]any{
			"capture": map[string]any{
				"supported":      true,
				"mode":           ternary(req.Target.PersistentMount, "log-dir-capture", "degraded-emergency"),
				"captureCommand": "enumerate broker log dirs and metadata",
				"artifactFile":   "backup.kafka.json",
				"steps":          []string{"inspect broker log dirs", "capture broker metadata listing", "seal checksum and manifest bundle"},
			},
			"restore": map[string]any{
				"supported":          true,
				"requiredArtifactID": "kafka-<service>-<timestamp>",
				"expectedFiles":      []string{"manifest.json", "checksum.sha256", "capture-plan.json", "restore.sh", "restore.ps1", "execution.json"},
				"restoreCommand":     "restore broker log directories before cluster rejoin",
			},
		},
		"strategy": map[string]any{
			"primary":              ternary(req.Target.PersistentMount, "broker log-dir protection with durable storage expectations", "container-layer emergency export"),
			"restoreValidation":    ternary(req.Target.PersistentMount, "restore-test eligible", "integrity only until durable log mount exists"),
			"replicationSuggested": true,
		},
		"data": map[string]any{
			"mount":              dataMount,
			"persistentMount":    req.Target.PersistentMount,
			"brokerType":         brokerType,
			"expectedPaths":      []string{"meta.properties", "log.dirs", "__consumer_offsets"},
			"validationMode":     req.ValidationMode,
			"restoreTestPolicy":  req.RestoreTestPolicy,
			"degradedProtection": record.Degraded,
		},
		"commands": map[string]any{
			"integrityCheck": "verify manifest + broker log-dir posture",
			"restoreHint":    "restore broker log directories before cluster rejoin",
			"operatorHint":   "message durability depends on broker config and replication, not only local data files",
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
	if manifest["serviceType"] != "kafka" {
		return sdk.ValidationResult{}, fmt.Errorf("artifact %s is not a kafka manifest", artifact.ID)
	}
	execution, _ := manifest["execution"].(map[string]any)
	restore, _ := execution["restore"].(map[string]any)
	if restore == nil {
		return sdk.ValidationResult{}, fmt.Errorf("artifact %s is missing kafka restore metadata", artifact.ID)
	}
	data, _ := manifest["data"].(map[string]any)
	persistent, _ := data["persistentMount"].(bool)

	return sdk.ValidationResult{
		IntegrityOK: true,
		RestoreTest: persistent,
		Degraded:    !persistent,
		Reason:      ternary(persistent, "kafka manifest verified with persistent broker storage", "kafka manifest verified in degraded emergency mode"),
	}, nil
}

func (Adapter) Restore(_ context.Context, req sdk.RestoreRequest) (sdk.RestoreResult, error) {
	if req.ArtifactID == "" || !strings.HasPrefix(req.ArtifactID, "kafka-") {
		return sdk.RestoreResult{}, fmt.Errorf("artifact id %q is not a kafka-generated bundle", req.ArtifactID)
	}
	if req.ArtifactPath == "" {
		return sdk.RestoreResult{}, fmt.Errorf("artifact id %q is missing manifest path", req.ArtifactID)
	}
	payload, err := adapterutil.ReadArtifactManifest(req.ArtifactPath)
	if err != nil {
		return sdk.RestoreResult{}, err
	}
	manifest, _ := payload["manifest"].(map[string]any)
	if serviceType, _ := manifest["serviceType"].(string); serviceType != "kafka" {
		return sdk.RestoreResult{}, fmt.Errorf("artifact id %q does not contain a kafka manifest", req.ArtifactID)
	}
	if req.BundleDir == "" {
		return sdk.RestoreResult{}, fmt.Errorf("artifact id %q is missing bundle path", req.ArtifactID)
	}
	for _, name := range []string{"manifest.json", "checksum.sha256", "capture-plan.json", "restore.sh", "restore.ps1", "execution.json"} {
		if _, err := os.Stat(filepath.Join(req.BundleDir, name)); err != nil {
			return sdk.RestoreResult{}, fmt.Errorf("artifact id %q is missing bundle file %s: %w", req.ArtifactID, name, err)
		}
	}
	return sdk.RestoreResult{Recovered: true, Details: map[string]any{"bundleDir": req.BundleDir, "mode": "bundle-verified-kafka-restore"}}, nil
}

func ternary[T any](condition bool, whenTrue, whenFalse T) T {
	if condition {
		return whenTrue
	}
	return whenFalse
}
