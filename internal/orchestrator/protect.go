package orchestrator

import (
	"context"
	"fmt"

	"github.com/kusuridheeraj/stateguard/internal/adapterutil"
	"github.com/kusuridheeraj/stateguard/internal/artifacts"
	"github.com/kusuridheeraj/stateguard/internal/backupexec"
	"github.com/kusuridheeraj/stateguard/internal/compose"
	"github.com/kusuridheeraj/stateguard/pkg/sdk"
	"github.com/kusuridheeraj/stateguard/pkg/types"
)

type Protector struct {
	store           *artifacts.Store
	registry        *sdk.Registry
	composeExecutor backupexec.ComposeExecutor
}

type ProtectReport struct {
	Project   string                 `json:"project" yaml:"project"`
	Runtime   string                 `json:"runtime" yaml:"runtime"`
	Created   int                    `json:"created" yaml:"created"`
	Artifacts []types.ArtifactRecord `json:"artifacts" yaml:"artifacts"`
	Skipped   []string               `json:"skipped" yaml:"skipped"`
	ScopeHint types.ProtectedScope   `json:"scopeHint" yaml:"scopeHint"`
}

type RestoreReport struct {
	ArtifactID string               `json:"artifactId" yaml:"artifactId"`
	Adapter    string               `json:"adapter" yaml:"adapter"`
	Recovered  bool                 `json:"recovered" yaml:"recovered"`
	Details    map[string]any       `json:"details,omitempty" yaml:"details,omitempty"`
	Record     types.ArtifactRecord `json:"record" yaml:"record"`
}

func NewProtector(store *artifacts.Store, registry *sdk.Registry) *Protector {
	return &Protector{
		store:           store,
		registry:        registry,
		composeExecutor: backupexec.NewComposeExecutor(),
	}
}

func (p *Protector) SetComposeLiveExecution(enabled bool) {
	p.composeExecutor.ExecuteLive = enabled
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
		if project.Runtime == "compose" {
			execResult, execErr := p.composeExecutor.ExecuteProtection(ctx, path, target, result.Manifest, record.BundleDir)
			record.SizeBytes += execResult.BytesWritten
			if execErr != nil {
				return ProtectReport{}, execErr
			}
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

func (p *Protector) RestoreArtifact(ctx context.Context, artifactID string) (RestoreReport, error) {
	record, ok := p.store.GetByID(artifactID)
	if !ok {
		return RestoreReport{}, fmt.Errorf("artifact %s not found", artifactID)
	}

	payload, err := adapterutil.ReadArtifactManifest(record.Path)
	if err != nil {
		return RestoreReport{}, fmt.Errorf("read artifact manifest: %w", err)
	}

	adapterName, _ := payload["adapter"].(string)
	if adapterName == "" {
		return RestoreReport{}, fmt.Errorf("artifact %s is missing adapter metadata", artifactID)
	}

	adapter, ok := p.registry.GetByName(adapterName)
	if !ok {
		return RestoreReport{}, fmt.Errorf("adapter %s is not registered", adapterName)
	}

	result, err := adapter.Restore(ctx, sdk.RestoreRequest{
		ArtifactID:   record.ID,
		ArtifactPath: record.Path,
		BundleDir:    record.BundleDir,
	})
	if err != nil {
		return RestoreReport{}, err
	}

	return RestoreReport{
		ArtifactID: record.ID,
		Adapter:    adapterName,
		Recovered:  result.Recovered,
		Details:    result.Details,
		Record:     record,
	}, nil
}
