package sdk

import (
	"context"
	"strings"

	"github.com/kusuridheeraj/stateguard/pkg/types"
)

type Adapter interface {
	Name() string
	Metadata() Metadata
	Detect(ctx context.Context, target Target) (DetectionResult, error)
	Protect(ctx context.Context, req ProtectRequest) (ProtectResult, error)
	Validate(ctx context.Context, artifact ArtifactRef) (ValidationResult, error)
	Restore(ctx context.Context, req RestoreRequest) (RestoreResult, error)
}

type Metadata struct {
	Official     bool
	Priority     int
	ImageHints   []string
	Description  string
	Capabilities []string
}

type Target struct {
	Name            string
	Scope           string
	Runtime         string
	Identifier      string
	Image           string
	Mounts          []string
	PersistentMount bool
	StatefulHint    bool
}

type DetectionResult struct {
	Supported  bool
	Stateful   bool
	Confidence string
	Reason     string
}

type ProtectRequest struct {
	Target            Target
	ArtifactRoot      string
	ValidationMode    string
	RestoreTestPolicy string
}

type ProtectResult struct {
	Record   types.ArtifactRecord
	Manifest map[string]any
}

type ArtifactRef struct {
	ID   string
	Path string
}

type ValidationResult struct {
	IntegrityOK bool
	RestoreTest bool
	Degraded    bool
	Reason      string
}

type RestoreRequest struct {
	ArtifactID string
}

type RestoreResult struct {
	Recovered bool
}

func MatchesImage(target Target, hints []string) bool {
	image := strings.ToLower(target.Image)
	for _, hint := range hints {
		if strings.Contains(image, strings.ToLower(hint)) {
			return true
		}
	}
	return false
}
