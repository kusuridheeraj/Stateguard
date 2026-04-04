package adapterutil

import (
	"crypto/sha256"
	"encoding/hex"
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

func WriteArtifactBundle(root, adapterName string, record types.ArtifactRecord, manifest map[string]any) (types.ArtifactRecord, error) {
	bundleDir := filepath.Join(root, sanitize(record.Scope), sanitize(record.Service), record.ID)
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return types.ArtifactRecord{}, fmt.Errorf("create bundle dir: %w", err)
	}

	manifestPayload := map[string]any{
		"record":   record,
		"adapter":  adapterName,
		"manifest": manifest,
	}
	manifestPath := filepath.Join(bundleDir, "manifest.json")
	manifestContent, err := json.MarshalIndent(manifestPayload, "", "  ")
	if err != nil {
		return types.ArtifactRecord{}, fmt.Errorf("encode artifact manifest: %w", err)
	}
	if err := os.WriteFile(manifestPath, manifestContent, 0o600); err != nil {
		return types.ArtifactRecord{}, fmt.Errorf("write artifact manifest: %w", err)
	}

	checksum := sha256.Sum256(manifestContent)
	checksumHex := hex.EncodeToString(checksum[:])
	checksumPath := filepath.Join(bundleDir, "checksum.sha256")
	checksumContent := []byte(fmt.Sprintf("%s  manifest.json\n", checksumHex))
	if err := os.WriteFile(checksumPath, checksumContent, 0o600); err != nil {
		return types.ArtifactRecord{}, fmt.Errorf("write checksum: %w", err)
	}

	capturePlan := buildCapturePlan(adapterName, record, manifest)
	planPath := filepath.Join(bundleDir, "capture-plan.json")
	planContent, err := json.MarshalIndent(capturePlan, "", "  ")
	if err != nil {
		return types.ArtifactRecord{}, fmt.Errorf("encode capture plan: %w", err)
	}
	if err := os.WriteFile(planPath, planContent, 0o600); err != nil {
		return types.ArtifactRecord{}, fmt.Errorf("write capture plan: %w", err)
	}

	restoreShell := []byte(renderRestoreScript(false, record, manifest))
	restorePowerShell := []byte(renderRestoreScript(true, record, manifest))
	restoreShPath := filepath.Join(bundleDir, "restore.sh")
	restorePSPath := filepath.Join(bundleDir, "restore.ps1")
	if err := os.WriteFile(restoreShPath, restoreShell, 0o700); err != nil {
		return types.ArtifactRecord{}, fmt.Errorf("write restore.sh: %w", err)
	}
	if err := os.WriteFile(restorePSPath, restorePowerShell, 0o600); err != nil {
		return types.ArtifactRecord{}, fmt.Errorf("write restore.ps1: %w", err)
	}

	record.BundleDir = bundleDir
	record.Path = manifestPath
	record.ChecksumSHA256 = checksumHex
	record.SizeBytes = int64(len(manifestContent) + len(checksumContent) + len(planContent) + len(restoreShell) + len(restorePowerShell))
	return record, nil
}

func buildCapturePlan(adapterName string, record types.ArtifactRecord, manifest map[string]any) map[string]any {
	strategy, _ := manifest["strategy"].(map[string]any)
	commands, _ := manifest["commands"].(map[string]any)
	data, _ := manifest["data"].(map[string]any)

	return map[string]any{
		"adapter":   adapterName,
		"artifact":  record.ID,
		"scope":     record.Scope,
		"service":   record.Service,
		"runtime":   record.Runtime,
		"degraded":  record.Degraded,
		"strategy":  strategy,
		"commands":  commands,
		"data":      data,
		"generated": record.CreatedAt,
	}
}

func renderRestoreScript(powershell bool, record types.ArtifactRecord, manifest map[string]any) string {
	commands, _ := manifest["commands"].(map[string]any)
	restoreHint, _ := commands["restoreHint"].(string)
	if restoreHint == "" {
		restoreHint = "follow adapter-specific restore guidance from manifest.json"
	}

	if powershell {
		return fmt.Sprintf(`# Stateguard restore helper for %s/%s
$ErrorActionPreference = "Stop"
$BundleDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Write-Host "Restore bundle: %s"
Write-Host "Manifest: $BundleDir\manifest.json"
Write-Host "Checksum: $BundleDir\checksum.sha256"
Write-Host "Hint: %s"
`, record.Scope, record.Service, record.ID, restoreHint)
	}

	return fmt.Sprintf(`#!/usr/bin/env sh
set -eu
BUNDLE_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
echo "Restore bundle: %s"
echo "Manifest: $BUNDLE_DIR/manifest.json"
echo "Checksum: $BUNDLE_DIR/checksum.sha256"
echo "Hint: %s"
`, record.ID, restoreHint)
}

func sanitize(value string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-")
	return replacer.Replace(value)
}
