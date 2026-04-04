package orchestrator

import (
	"context"
	"fmt"

	"github.com/kusuridheeraj/stateguard/internal/adapterutil"
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
	record, err := adapterutil.WriteArtifactBundle(p.store.Root(), adapterName, result.Record, result.Manifest)
	if err != nil {
		return types.ArtifactRecord{}, fmt.Errorf("persist artifact bundle: %w", err)
	}
	return record, nil
}
