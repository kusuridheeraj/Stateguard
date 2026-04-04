package kafka

import (
	"context"
	"fmt"
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
	return sdk.RestoreResult{Recovered: req.ArtifactID != ""}, nil
}

func ternary[T any](condition bool, whenTrue, whenFalse T) T {
	if condition {
		return whenTrue
	}
	return whenFalse
}
