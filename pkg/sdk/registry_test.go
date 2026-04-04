package sdk

import (
	"context"
	"testing"
)

type fakeAdapter struct {
	name string
	meta Metadata
}

func (f fakeAdapter) Name() string       { return f.name }
func (f fakeAdapter) Metadata() Metadata { return f.meta }
func (f fakeAdapter) Detect(_ context.Context, target Target) (DetectionResult, error) {
	if MatchesImage(target, f.meta.ImageHints) {
		return DetectionResult{Supported: true, Stateful: true, Confidence: "high"}, nil
	}
	return DetectionResult{}, nil
}
func (f fakeAdapter) Protect(context.Context, ProtectRequest) (ProtectResult, error) {
	return ProtectResult{}, nil
}
func (f fakeAdapter) Validate(context.Context, ArtifactRef) (ValidationResult, error) {
	return ValidationResult{}, nil
}
func (f fakeAdapter) Restore(context.Context, RestoreRequest) (RestoreResult, error) {
	return RestoreResult{}, nil
}

func TestRegistryResolveByPriority(t *testing.T) {
	registry := NewRegistry(
		fakeAdapter{name: "generic", meta: Metadata{Priority: 1, ImageHints: []string{"postgres"}}},
		fakeAdapter{name: "postgres", meta: Metadata{Priority: 10, ImageHints: []string{"postgres"}}},
	)

	adapter, _, ok := registry.Resolve(context.Background(), Target{Image: "postgres:16"})
	if !ok {
		t.Fatal("expected adapter resolution to succeed")
	}
	if adapter.Name() != "postgres" {
		t.Fatalf("expected highest priority adapter, got %q", adapter.Name())
	}
}
