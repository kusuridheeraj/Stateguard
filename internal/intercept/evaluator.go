package intercept

import (
	"context"
	"fmt"

	"github.com/kusuridheeraj/stateguard/internal/orchestrator"
)

type Operation string

const (
	OpComposeDown            Operation = "compose.down"
	OpComposeDownWithVolumes Operation = "compose.down.volumes"
	OpDockerVolumeRemove     Operation = "docker.volume.remove"
	OpDockerSystemPrune      Operation = "docker.system.prune"
)

type ProtectFunc func(context.Context, string) (orchestrator.ProtectReport, error)

type Evaluator struct {
	Mode           string
	ProtectCompose ProtectFunc
}

type Result struct {
	Allowed       bool                       `json:"allowed" yaml:"allowed"`
	Operation     Operation                  `json:"operation" yaml:"operation"`
	ProtectionRun bool                       `json:"protectionRun" yaml:"protectionRun"`
	Reason        string                     `json:"reason" yaml:"reason"`
	Protection    orchestrator.ProtectReport `json:"protection" yaml:"protection"`
}

func (e Evaluator) EvaluateComposeOperation(ctx context.Context, composePath string, operation Operation) (Result, error) {
	if e.ProtectCompose == nil {
		return Result{}, fmt.Errorf("protect compose callback is required")
	}

	report, err := e.ProtectCompose(ctx, composePath)
	if err != nil {
		return Result{
			Allowed:       false,
			Operation:     operation,
			ProtectionRun: true,
			Reason:        fmt.Sprintf("preflight protection failed: %v", err),
		}, err
	}

	allowed := report.Created > 0 || len(report.ScopeHint.Name) > 0
	reason := "preflight protection completed"
	if len(report.Skipped) > 0 {
		reason = fmt.Sprintf("%s; skipped services: %v", reason, report.Skipped)
	}

	switch operation {
	case OpComposeDownWithVolumes, OpDockerVolumeRemove, OpDockerSystemPrune:
		reason = fmt.Sprintf("%s; destructive storage impact expected", reason)
	}

	return Result{
		Allowed:       allowed,
		Operation:     operation,
		ProtectionRun: true,
		Reason:        reason,
		Protection:    report,
	}, nil
}
