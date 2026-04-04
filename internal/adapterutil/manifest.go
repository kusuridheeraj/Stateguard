package adapterutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kusuridheeraj/stateguard/pkg/sdk"
	"github.com/kusuridheeraj/stateguard/pkg/types"
)

func NewRecord(adapterName string, req sdk.ProtectRequest, strongValidation bool) types.ArtifactRecord {
	return types.ArtifactRecord{
		ID:                 fmt.Sprintf("%s-%s-%d", adapterName, req.Target.Name, time.Now().UTC().UnixNano()),
		Scope:              req.Target.Scope,
		Service:            req.Target.Name,
		Runtime:            req.Target.Runtime,
		CreatedAt:          time.Now().UTC(),
		IntegrityValidated: true,
		RestoreTested:      strongValidation,
		Degraded:           !req.Target.PersistentMount,
	}
}

func MountForTarget(target sdk.Target, hints ...string) string {
	for _, mount := range target.Mounts {
		parts := strings.SplitN(mount, ":", 2)
		if len(parts) != 2 {
			continue
		}
		containerPath := parts[1]
		for _, hint := range hints {
			if strings.Contains(containerPath, hint) {
				return mount
			}
		}
	}
	if len(target.Mounts) > 0 {
		return target.Mounts[0]
	}
	return ""
}

func ReadArtifactManifest(path string) (map[string]any, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read artifact manifest: %w", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(content, &payload); err != nil {
		return nil, fmt.Errorf("decode artifact manifest: %w", err)
	}
	return payload, nil
}

func WriteManifestPreview(root, adapterName, serviceName string, payload map[string]any) (string, int64, error) {
	dir := filepath.Join(root, "previews", adapterName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", 0, fmt.Errorf("create preview dir: %w", err)
	}

	path := filepath.Join(dir, fmt.Sprintf("%s.json", serviceName))
	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", 0, fmt.Errorf("encode preview payload: %w", err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return "", 0, fmt.Errorf("write preview payload: %w", err)
	}
	return path, int64(len(content)), nil
}
