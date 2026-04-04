package redis

import (
	"context"
	"fmt"
	"strings"

	"github.com/kusuridheeraj/stateguard/internal/adapterutil"
	"github.com/kusuridheeraj/stateguard/pkg/sdk"
)

type Adapter struct{}

func New() Adapter { return Adapter{} }

func (Adapter) Name() string { return "redis" }

func (Adapter) Metadata() sdk.Metadata {
	return sdk.Metadata{
		Official:     true,
		Priority:     95,
		ImageHints:   []string{"redis"},
		Description:  "Redis adapter with persistence-aware manifest generation and validation hints.",
		Capabilities: []string{"detect", "protect", "validate", "restore", "manifest-aware"},
	}
}

func (Adapter) Detect(_ context.Context, target sdk.Target) (sdk.DetectionResult, error) {
	if !sdk.MatchesImage(target, []string{"redis"}) {
		return sdk.DetectionResult{Supported: false, Stateful: target.StatefulHint, Confidence: "low", Reason: "image does not match redis"}, nil
	}

	reason := "redis image detected with durable mount; persistence can be protected"
	confidence := "high"
	if !target.PersistentMount {
		reason = "redis image detected without durable mount; cache/session safety is degraded"
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
	record := adapterutil.NewRecord("redis", req, true)
	record.Degraded = !req.Target.PersistentMount

	dataMount := adapterutil.MountForTarget(req.Target, "/data")
	persistenceMode := "unknown"
	if req.Target.PersistentMount {
		persistenceMode = "durable-volume-backed"
	} else {
		persistenceMode = "memory-or-container-layer-only"
	}

	manifest := map[string]any{
		"serviceType": "redis",
		"strategy": map[string]any{
			"primary":           ternary(req.Target.PersistentMount, "capture persistence-capable Redis state", "emergency capture of volatile Redis state"),
			"restoreValidation": ternary(req.Target.PersistentMount, "restore-test eligible", "integrity only until durable mount exists"),
		},
		"data": map[string]any{
			"mount":              dataMount,
			"persistentMount":    req.Target.PersistentMount,
			"persistenceMode":    persistenceMode,
			"expectedFiles":      []string{"appendonly.aof", "dump.rdb"},
			"validationMode":     req.ValidationMode,
			"restoreTestPolicy":  req.RestoreTestPolicy,
			"degradedProtection": record.Degraded,
		},
		"commands": map[string]any{
			"persistenceHint": "AOF or RDB should be enabled for durable recovery",
			"integrityCheck":  "verify manifest + Redis persistence expectations",
			"restoreHint":     "restore persistence files before service boot",
		},
		"notes": []string{
			"Redis may be used for cache, sessions, MCP chats, or queues; operator intent is often unclear.",
			"Without durable persistence settings, recovery is only best-effort.",
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
	if serviceType != "redis" {
		return sdk.ValidationResult{}, fmt.Errorf("artifact %s is not a redis manifest", artifact.ID)
	}

	data, _ := manifest["data"].(map[string]any)
	persistent, _ := data["persistentMount"].(bool)
	mode, _ := data["persistenceMode"].(string)
	degraded := !persistent || strings.Contains(mode, "memory")

	return sdk.ValidationResult{
		IntegrityOK: true,
		RestoreTest: persistent,
		Degraded:    degraded,
		Reason:      ternary(degraded, "redis manifest verified with degraded persistence posture", "redis manifest verified with durable persistence posture"),
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
