package policy

import (
	"fmt"

	"github.com/kusuridheeraj/stateguard/internal/config"
	"github.com/kusuridheeraj/stateguard/pkg/types"
)

type Engine struct {
	config config.PolicyConfig
}

func NewEngine(cfg config.PolicyConfig) Engine {
	return Engine{config: cfg}
}

func (e Engine) Evaluate(state types.ProtectionState) types.PolicyDecision {
	if !state.RecoveryPointExists {
		return block("error", "no recovery point exists")
	}
	if e.config.Validation.IntegrityCheck && !state.IntegrityValidated {
		return block("error", "recovery point integrity is not validated")
	}
	if e.config.Validation.RestoreTest.Enabled && !state.RestoreTested {
		return block("error", "recovery point has not passed restore validation")
	}
	if state.Degraded && !e.config.Validation.AllowDegraded {
		return block("warn", "protection is degraded and degraded mode is not allowed")
	}

	reason := "verified recovery point satisfies current policy"
	if state.Degraded {
		reason = fmt.Sprintf("%s; degraded mode explicitly allowed", reason)
	}

	return types.PolicyDecision{
		Allow:    true,
		Severity: "info",
		Reason:   reason,
	}
}

func block(severity, reason string) types.PolicyDecision {
	return types.PolicyDecision{
		Allow:    false,
		Severity: severity,
		Reason:   reason,
	}
}
