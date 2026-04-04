package scheduler

import (
	"context"
	"testing"
	"time"
)

func TestSchedulerRunOnce(t *testing.T) {
	s := New()
	ran := false
	s.Register("demo", time.Minute, func(context.Context) error {
		ran = true
		return nil
	})

	if err := s.RunOnce(context.Background(), "demo"); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if !ran {
		t.Fatal("expected scheduled job to run")
	}

	snapshot := s.Snapshot()
	if len(snapshot) != 1 || snapshot[0].LastSuccessAt.IsZero() {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}
}
