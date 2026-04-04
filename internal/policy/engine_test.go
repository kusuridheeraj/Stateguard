package policy

import (
	"testing"
	"time"

	"github.com/kusuridheeraj/stateguard/internal/config"
	"github.com/kusuridheeraj/stateguard/pkg/types"
)

func TestEvaluateAllowsVerifiedProtection(t *testing.T) {
	engine := NewEngine(config.Default().Policy)
	decision := engine.Evaluate(types.ProtectionState{
		RecoveryPointExists: true,
		IntegrityValidated:  true,
		RestoreTested:       true,
		LastValidatedAt:     time.Now(),
	})
	if !decision.Allow {
		t.Fatalf("expected allow decision, got %#v", decision)
	}
}

func TestEvaluateBlocksWhenRecoveryPointMissing(t *testing.T) {
	engine := NewEngine(config.Default().Policy)
	decision := engine.Evaluate(types.ProtectionState{})
	if decision.Allow {
		t.Fatalf("expected block decision, got %#v", decision)
	}
}
