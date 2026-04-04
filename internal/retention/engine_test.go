package retention

import (
	"testing"
	"time"

	"github.com/kusuridheeraj/stateguard/pkg/types"
)

func TestEvaluateRemovesExpiredArtifacts(t *testing.T) {
	engine := NewEngine(24 * time.Hour)
	now := time.Now().UTC()
	plan := engine.Evaluate([]types.ArtifactRecord{
		{ID: "old", CreatedAt: now.Add(-48 * time.Hour), SizeBytes: 10},
		{ID: "new", CreatedAt: now, SizeBytes: 10},
	}, Snapshot{}, 20, now)

	if len(plan.DeleteIDs) != 1 || plan.DeleteIDs[0] != "old" {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}

func TestEvaluateRemovesOldestWhenQuotaExceeded(t *testing.T) {
	engine := NewEngine(0)
	now := time.Now().UTC()
	plan := engine.Evaluate([]types.ArtifactRecord{
		{ID: "old", CreatedAt: now.Add(-2 * time.Hour), SizeBytes: 50},
		{ID: "new", CreatedAt: now, SizeBytes: 50},
	}, Snapshot{CapacityBytes: 100, UsedBytes: 90}, 50, now)

	if len(plan.DeleteIDs) != 1 || plan.DeleteIDs[0] != "old" {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}
