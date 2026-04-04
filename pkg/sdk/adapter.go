package sdk

import "context"

type Adapter interface {
	Name() string
	Detect(ctx context.Context, target Target) (DetectionResult, error)
	Protect(ctx context.Context, req ProtectRequest) (ProtectResult, error)
	Validate(ctx context.Context, artifact ArtifactRef) (ValidationResult, error)
	Restore(ctx context.Context, req RestoreRequest) (RestoreResult, error)
}

type Target struct {
	Name       string
	Runtime    string
	Identifier string
}

type DetectionResult struct {
	Stateful bool
	Reason   string
}

type ProtectRequest struct {
	Target Target
}

type ProtectResult struct {
	ArtifactID string
}

type ArtifactRef struct {
	ID string
}

type ValidationResult struct {
	IntegrityOK bool
	RestoreTest bool
}

type RestoreRequest struct {
	ArtifactID string
}

type RestoreResult struct {
	Recovered bool
}
