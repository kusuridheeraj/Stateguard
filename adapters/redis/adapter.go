package redis

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
	persistenceMode := redisPersistenceMode(req.Target.PersistentMount)
	backupMode := redisBackupMode(req.Target.PersistentMount)
	restoreMode := redisRestoreMode(req.Target.PersistentMount)
	expectedFiles := []string{"appendonly.aof", "dump.rdb"}

	manifest := map[string]any{
		"serviceType": "redis",
		"execution": map[string]any{
			"capture": map[string]any{
				"supported":           true,
				"mode":                backupMode,
				"source":              ternary(req.Target.PersistentMount, "durable Redis persistence files", "live container memory + persistence posture"),
				"steps":               []string{"inspect persistence configuration", "capture persistence files or emergency snapshot", "write checksum and capture plan", "stage restore helper scripts"},
				"captureCommand":      ternary(req.Target.PersistentMount, "redis-cli --rdb /bundle/dump.rdb", "emergency snapshot capture"),
				"fallbackCommand":     "redis-cli SAVE",
				"artifactLayout":      []string{"manifest.json", "checksum.sha256", "capture-plan.json", "restore.sh", "restore.ps1"},
				"restoreArtifactHint": "bundle can be replayed from manifest.json and checksum.sha256",
			},
			"restore": map[string]any{
				"supported":          true,
				"mode":               restoreMode,
				"requiredArtifactID": "redis-<service>-<timestamp>",
				"expectedFiles":      expectedFiles,
				"steps":              []string{"verify generated artifact id", "load manifest and checksum metadata", "restore persistence files", "restart or reattach Redis service"},
				"validationMode":     req.ValidationMode,
				"restoreTestPolicy":  req.RestoreTestPolicy,
			},
		},
		"artifact": map[string]any{
			"id":          record.ID,
			"bundleFiles": []string{"manifest.json", "checksum.sha256", "capture-plan.json", "restore.sh", "restore.ps1", "execution.json"},
		},
		"strategy": map[string]any{
			"primary":           ternary(req.Target.PersistentMount, "capture persistence-capable Redis state with live persistence files", "emergency capture of volatile Redis state with degraded semantics"),
			"restoreValidation": ternary(req.Target.PersistentMount, "restore-test eligible", "integrity only until durable mount exists"),
		},
		"data": map[string]any{
			"mount":              dataMount,
			"persistentMount":    req.Target.PersistentMount,
			"persistenceMode":    persistenceMode,
			"expectedFiles":      expectedFiles,
			"validationMode":     req.ValidationMode,
			"restoreTestPolicy":  req.RestoreTestPolicy,
			"degradedProtection": record.Degraded,
		},
		"commands": map[string]any{
			"persistenceHint": "AOF or RDB should be enabled for durable recovery",
			"integrityCheck":  "verify manifest + Redis persistence expectations",
			"backupCommand":   redisBackupCommand(req.Target.PersistentMount),
			"restoreHint":     redisRestoreHint(req.Target.PersistentMount),
			"restoreCommand":  redisRestoreCommand(req.Target.PersistentMount),
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

	execution, ok := manifest["execution"].(map[string]any)
	if !ok {
		return sdk.ValidationResult{}, fmt.Errorf("artifact %s is missing redis execution metadata", artifact.ID)
	}
	restore, ok := execution["restore"].(map[string]any)
	if !ok {
		return sdk.ValidationResult{}, fmt.Errorf("artifact %s is missing redis restore metadata", artifact.ID)
	}

	data, _ := manifest["data"].(map[string]any)
	persistent, _ := data["persistentMount"].(bool)
	mode, _ := data["persistenceMode"].(string)
	restoreSupported, _ := restore["supported"].(bool)
	expectedFiles, _ := restore["expectedFiles"].([]any)
	degraded := !persistent || strings.Contains(mode, "memory") || !restoreSupported || len(expectedFiles) == 0

	return sdk.ValidationResult{
		IntegrityOK: true,
		RestoreTest: persistent,
		Degraded:    degraded,
		Reason:      ternary(degraded, "redis manifest verified with degraded persistence posture", "redis manifest verified with durable persistence posture"),
	}, nil
}

func (Adapter) Restore(_ context.Context, req sdk.RestoreRequest) (sdk.RestoreResult, error) {
	if !isRedisArtifactID(req.ArtifactID) {
		return sdk.RestoreResult{}, fmt.Errorf("artifact id %q is not a redis-generated recovery bundle", req.ArtifactID)
	}
	if req.ArtifactPath == "" {
		return sdk.RestoreResult{}, fmt.Errorf("artifact id %q is missing manifest path", req.ArtifactID)
	}
	payload, err := adapterutil.ReadArtifactManifest(req.ArtifactPath)
	if err != nil {
		return sdk.RestoreResult{}, err
	}
	manifest, _ := payload["manifest"].(map[string]any)
	if serviceType, _ := manifest["serviceType"].(string); serviceType != "redis" {
		return sdk.RestoreResult{}, fmt.Errorf("artifact id %q does not contain a redis manifest", req.ArtifactID)
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
			"mode":      "bundle-verified-redis-restore",
		},
	}, nil
}

func ternary[T any](condition bool, whenTrue, whenFalse T) T {
	if condition {
		return whenTrue
	}
	return whenFalse
}

func redisPersistenceMode(persistent bool) string {
	if persistent {
		return "durable-volume-backed"
	}
	return "memory-or-container-layer-only"
}

func redisBackupMode(persistent bool) string {
	if persistent {
		return "volume-snapshot + redis persistence file capture"
	}
	return "emergency live container snapshot"
}

func redisRestoreMode(persistent bool) string {
	if persistent {
		return "persistence-file replay + service restart"
	}
	return "best-effort emergency restore"
}

func redisBackupCommand(persistent bool) string {
	if persistent {
		return "redis-cli --rdb /bundle/dump.rdb"
	}
	return "capture live Redis container snapshot and stage persistence metadata"
}

func redisRestoreHint(persistent bool) string {
	if persistent {
		return "replay persistence files before bringing Redis back online"
	}
	return "restore the captured emergency bundle before service boot"
}

func redisRestoreCommand(persistent bool) string {
	if persistent {
		return "copy appendonly.aof or dump.rdb back to /data and restart Redis"
	}
	return "restore the emergency bundle and restart Redis in degraded mode"
}

func isRedisArtifactID(artifactID string) bool {
	if !strings.HasPrefix(artifactID, "redis-") {
		return false
	}
	lastDash := strings.LastIndex(artifactID, "-")
	if lastDash <= len("redis-") || lastDash == len(artifactID)-1 {
		return false
	}
	for _, r := range artifactID[lastDash+1:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
