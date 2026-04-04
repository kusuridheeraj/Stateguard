package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kusuridheeraj/stateguard/internal/artifacts"
	"github.com/kusuridheeraj/stateguard/internal/compose"
	"github.com/kusuridheeraj/stateguard/pkg/sdk"
	"github.com/kusuridheeraj/stateguard/pkg/types"
)

type Protector struct {
	store    *artifacts.Store
	registry *sdk.Registry
}

type ProtectReport struct {
	Project   string                 `json:"project" yaml:"project"`
	Runtime   string                 `json:"runtime" yaml:"runtime"`
	Created   int                    `json:"created" yaml:"created"`
	Artifacts []types.ArtifactRecord `json:"artifacts" yaml:"artifacts"`
	Skipped   []string               `json:"skipped" yaml:"skipped"`
	ScopeHint types.ProtectedScope   `json:"scopeHint" yaml:"scopeHint"`
}

func NewProtector(store *artifacts.Store, registry *sdk.Registry) *Protector {
	return &Protector{store: store, registry: registry}
}

func (p *Protector) ProtectCompose(ctx context.Context, path string) (ProtectReport, error) {
	project, err := compose.Discover(path)
	if err != nil {
		return ProtectReport{}, err
	}

	report := ProtectReport{
		Project:   project.Name,
		Runtime:   project.Runtime,
		Artifacts: []types.ArtifactRecord{},
		Skipped:   []string{},
		ScopeHint: project.ScopeHint,
	}

	for _, service := range project.Services {
		target := sdk.Target{
			Name:            service.Name,
			Scope:           project.Name,
			Runtime:         project.Runtime,
			Identifier:      fmt.Sprintf("%s/%s", project.Name, service.Name),
			Image:           service.Image,
			Mounts:          append([]string{}, service.Mounts...),
			PersistentMount: service.HasPersistentMount,
			StatefulHint:    service.StatefulCandidate,
		}

		adapter, _, ok := p.registry.Resolve(ctx, target)
		if !ok {
			report.Skipped = append(report.Skipped, service.Name)
			continue
		}

		result, err := adapter.Protect(ctx, sdk.ProtectRequest{
			Target:            target,
			ArtifactRoot:      p.store.Root(),
			ValidationMode:    "hybrid",
			RestoreTestPolicy: "periodic",
		})
		if err != nil {
			return ProtectReport{}, err
		}

		record, err := p.persistArtifact(adapter.Name(), result)
		if err != nil {
			return ProtectReport{}, err
		}

		validation, err := adapter.Validate(ctx, sdk.ArtifactRef{ID: record.ID, Path: record.Path})
		if err != nil {
			return ProtectReport{}, err
		}
		record.IntegrityValidated = validation.IntegrityOK
		record.RestoreTested = validation.RestoreTest
		record.Degraded = record.Degraded || validation.Degraded

		if err := p.store.Add(record); err != nil {
			return ProtectReport{}, err
		}

		report.Created++
		report.Artifacts = append(report.Artifacts, record)
	}

	return report, nil
}

func (p *Protector) persistArtifact(adapterName string, result sdk.ProtectResult) (types.ArtifactRecord, error) {
	record := result.Record
	serviceDir := filepath.Join(p.store.Root(), sanitize(record.Scope), sanitize(record.Service))
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		return types.ArtifactRecord{}, fmt.Errorf("create artifact dir: %w", err)
	}

	record.Path = filepath.Join(serviceDir, record.ID+".json")

	payload := map[string]any{
		"record":   record,
		"adapter":  adapterName,
		"manifest": result.Manifest,
	}

	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return types.ArtifactRecord{}, fmt.Errorf("encode artifact manifest: %w", err)
	}
	if err := os.WriteFile(record.Path, content, 0o600); err != nil {
		return types.ArtifactRecord{}, fmt.Errorf("write artifact manifest: %w", err)
	}
	record.SizeBytes = int64(len(content))
	return record, nil
}

func sanitize(value string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-")
	return replacer.Replace(value)
}
