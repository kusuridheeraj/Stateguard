package redis

import (
	"context"

	"github.com/kusuridheeraj/stateguard/internal/adapterutil"
	"github.com/kusuridheeraj/stateguard/pkg/sdk"
)

type Adapter struct{ scaffold adapterutil.Scaffold }

func New() Adapter {
	return Adapter{scaffold: adapterutil.Scaffold{
		AdapterName:      "redis",
		ImageHints:       []string{"redis"},
		Description:      "Redis adapter scaffold for protection, validation, and restore.",
		Official:         true,
		Priority:         95,
		Capabilities:     []string{"detect", "protect", "validate", "restore"},
		StrongValidation: true,
	}}
}

func (a Adapter) Name() string           { return a.scaffold.AdapterName }
func (a Adapter) Metadata() sdk.Metadata { return a.scaffold.Metadata() }
func (a Adapter) Detect(ctx context.Context, target sdk.Target) (sdk.DetectionResult, error) {
	return a.scaffold.Detect(ctx, target)
}
func (a Adapter) Protect(ctx context.Context, req sdk.ProtectRequest) (sdk.ProtectResult, error) {
	return a.scaffold.Protect(ctx, req)
}
func (a Adapter) Validate(ctx context.Context, artifact sdk.ArtifactRef) (sdk.ValidationResult, error) {
	return a.scaffold.Validate(ctx, artifact)
}
func (a Adapter) Restore(ctx context.Context, req sdk.RestoreRequest) (sdk.RestoreResult, error) {
	return a.scaffold.Restore(ctx, req)
}
