package adapterutil

import (
	"context"
	"fmt"
	"time"

	"github.com/kusuridheeraj/stateguard/pkg/sdk"
	"github.com/kusuridheeraj/stateguard/pkg/types"
)

type Scaffold struct {
	AdapterName      string
	ImageHints       []string
	Description      string
	Official         bool
	Priority         int
	Capabilities     []string
	StrongValidation bool
}

func (s Scaffold) Metadata() sdk.Metadata {
	return sdk.Metadata{
		Official:     s.Official,
		Priority:     s.Priority,
		ImageHints:   append([]string{}, s.ImageHints...),
		Description:  s.Description,
		Capabilities: append([]string{}, s.Capabilities...),
	}
}

func (s Scaffold) Detect(_ context.Context, target sdk.Target) (sdk.DetectionResult, error) {
	if sdk.MatchesImage(target, s.ImageHints) {
		return sdk.DetectionResult{
			Supported:  true,
			Stateful:   true,
			Confidence: "high",
			Reason:     fmt.Sprintf("matched adapter image hints for %s", s.AdapterName),
		}, nil
	}
	return sdk.DetectionResult{
		Supported:  false,
		Stateful:   target.StatefulHint,
		Confidence: "low",
		Reason:     "image hints did not match",
	}, nil
}

func (s Scaffold) Protect(_ context.Context, req sdk.ProtectRequest) (sdk.ProtectResult, error) {
	id := fmt.Sprintf("%s-%s-%d", s.AdapterName, req.Target.Name, time.Now().UTC().UnixNano())
	record := types.ArtifactRecord{
		ID:                 id,
		Scope:              req.Target.Scope,
		Service:            req.Target.Name,
		Runtime:            req.Target.Runtime,
		CreatedAt:          time.Now().UTC(),
		IntegrityValidated: true,
		RestoreTested:      s.StrongValidation,
		Degraded:           !req.Target.PersistentMount,
	}
	manifest := map[string]any{
		"adapter":           s.AdapterName,
		"target":            req.Target,
		"validationMode":    req.ValidationMode,
		"restoreTestPolicy": req.RestoreTestPolicy,
		"strongValidation":  s.StrongValidation,
	}
	return sdk.ProtectResult{
		Record:   record,
		Manifest: manifest,
	}, nil
}

func (s Scaffold) Validate(_ context.Context, artifact sdk.ArtifactRef) (sdk.ValidationResult, error) {
	return sdk.ValidationResult{
		IntegrityOK: true,
		RestoreTest: s.StrongValidation,
		Degraded:    false,
		Reason:      fmt.Sprintf("scaffold validation completed for %s", artifact.ID),
	}, nil
}

func (s Scaffold) Restore(_ context.Context, req sdk.RestoreRequest) (sdk.RestoreResult, error) {
	return sdk.RestoreResult{Recovered: req.ArtifactID != ""}, nil
}
