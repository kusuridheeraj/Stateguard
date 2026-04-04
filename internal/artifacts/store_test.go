package artifacts

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/kusuridheeraj/stateguard/pkg/types"
)

func TestStoreAddAndSummary(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "artifacts"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	err = store.Add(types.ArtifactRecord{
		ID:                 "artifact-1",
		Scope:              "demo",
		Service:            "postgres",
		Runtime:            "compose",
		SizeBytes:          128,
		CreatedAt:          time.Now().UTC(),
		IntegrityValidated: true,
		RestoreTested:      true,
	})
	if err != nil {
		t.Fatalf("add artifact: %v", err)
	}

	summary := store.Summary()
	if summary.Count != 1 || summary.TotalSizeBytes != 128 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
}

func TestStoreLatestByScope(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "artifacts"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	first := time.Now().UTC().Add(-time.Hour)
	second := time.Now().UTC()
	for _, record := range []types.ArtifactRecord{
		{ID: "a", Scope: "demo", CreatedAt: first},
		{ID: "b", Scope: "demo", CreatedAt: second},
	} {
		if err := store.Add(record); err != nil {
			t.Fatalf("add artifact: %v", err)
		}
	}

	latest, ok := store.LatestByScope("demo")
	if !ok || latest.ID != "b" {
		t.Fatalf("expected latest artifact b, got %#v", latest)
	}
}
