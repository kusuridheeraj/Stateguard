package generic

import (
	"context"

	"github.com/kusuridheeraj/stateguard/internal/adapterutil"
	"github.com/kusuridheeraj/stateguard/pkg/sdk"
)

type Adapter struct{ scaffold adapterutil.Scaffold }

func New() Adapter {
	return Adapter{scaffold: adapterutil.Scaffold{
		AdapterName:      "generic",
		ImageHints:       []string{},
		Description:      "Generic fallback adapter for unknown stateful workloads.",
		Official:         true,
		Priority:         1,
		Capabilities:     []string{"detect", "protect", "validate", "restore", "fallback"},
		StrongValidation: false,
	}}
}

func (a Adapter) Name() string           { return a.scaffold.AdapterName }
func (a Adapter) Metadata() sdk.Metadata { return a.scaffold.Metadata() }

func (a Adapter) Detect(_ context.Context, target sdk.Target) (sdk.DetectionResult, error) {
	if target.StatefulHint || target.PersistentMount {
		return sdk.DetectionResult{
			Supported:  true,
			Stateful:   true,
			Confidence: "medium",
			Reason:     "generic fallback selected for stateful or persistent service",
		}, nil
	}
	return sdk.DetectionResult{
		Supported:  false,
		Stateful:   false,
		Confidence: "low",
		Reason:     "generic fallback skipped because no stateful hint was present",
	}, nil
}

func (a Adapter) Protect(ctx context.Context, req sdk.ProtectRequest) (sdk.ProtectResult, error) {
	return a.scaffold.Protect(ctx, req)
}
func (a Adapter) Validate(ctx context.Context, artifact sdk.ArtifactRef) (sdk.ValidationResult, error) {
	result, err := a.scaffold.Validate(ctx, artifact)
	result.Degraded = true
	result.Reason = "generic fallback validation completed with degraded guarantees"
	return result, err
}
func (a Adapter) Restore(ctx context.Context, req sdk.RestoreRequest) (sdk.RestoreResult, error) {
	return a.scaffold.Restore(ctx, req)
}
