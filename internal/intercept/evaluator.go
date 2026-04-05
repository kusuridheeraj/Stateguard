package intercept

import (
	"context"
	"fmt"

	"github.com/kusuridheeraj/stateguard/internal/compose"
	"github.com/kusuridheeraj/stateguard/internal/orchestrator"
)

type Operation string

const (
	OpComposeDown            Operation = "compose.down"
	OpComposeDownWithVolumes Operation = "compose.down.volumes"
	OpComposeUp              Operation = "compose.up"
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
	Scope         string                     `json:"scope,omitempty" yaml:"scope,omitempty"`
	Targets       []string                   `json:"targets,omitempty" yaml:"targets,omitempty"`
	Flags         []string                   `json:"flags,omitempty" yaml:"flags,omitempty"`
	Warnings      []string                   `json:"warnings,omitempty" yaml:"warnings,omitempty"`
	ProtectionRun bool                       `json:"protectionRun" yaml:"protectionRun"`
	Reason        string                     `json:"reason" yaml:"reason"`
	Protection    orchestrator.ProtectReport `json:"protection" yaml:"protection"`
	Executed      bool                       `json:"executed" yaml:"executed"`
	RunResult     *compose.RunResult         `json:"runResult,omitempty" yaml:"runResult,omitempty"`
}

func (e Evaluator) EvaluateComposeOperation(ctx context.Context, plan DockerArgsPlan) (Result, error) {
	if e.ProtectCompose == nil {
		return Result{}, fmt.Errorf("protect compose callback is required")
	}

	report, err := e.ProtectCompose(ctx, plan.ComposePath)
	if err != nil {
		return Result{
			Allowed:       false,
			Operation:     plan.Operation,
			Scope:         "compose-project",
			Targets:       append([]string(nil), plan.Targets...),
			Flags:         append([]string(nil), plan.Flags...),
			ProtectionRun: true,
			Reason:        fmt.Sprintf("preflight protection failed: %v", err),
			Warnings:      []string{"compose preflight could not be completed"},
		}, err
	}

	allowed := report.Created > 0 || len(report.ScopeHint.Name) > 0
	reason := "preflight protection completed"
	if len(report.Skipped) > 0 {
		reason = fmt.Sprintf("%s; skipped services: %v", reason, report.Skipped)
	}

	warnings := []string(nil)
	switch plan.Operation {
	case OpComposeDownWithVolumes:
		reason = fmt.Sprintf("%s; destructive storage impact expected", reason)
		warnings = append(warnings, "compose volumes may be removed by the runtime")
	}

	return Result{
		Allowed:       allowed,
		Operation:     plan.Operation,
		Scope:         "compose-project",
		Targets:       append([]string(nil), plan.Targets...),
		Flags:         append([]string(nil), plan.Flags...),
		Warnings:      warnings,
		ProtectionRun: true,
		Reason:        reason,
		Protection:    report,
	}, nil
}

func (e Evaluator) EvaluateDockerArgs(ctx context.Context, plan DockerArgsPlan) (Result, error) {
	switch plan.Operation {
	case OpComposeDown, OpComposeDownWithVolumes, OpComposeUp:
		return e.EvaluateComposeOperation(ctx, plan)
	case OpDockerVolumeRemove:
		return Result{
			Allowed:       false,
			Operation:     plan.Operation,
			Scope:         "host-global",
			Targets:       append([]string(nil), plan.Targets...),
			Flags:         append([]string(nil), plan.Flags...),
			ProtectionRun: false,
			Reason:        "docker volume rm is a host-global destructive operation and cannot be auto-executed without explicit volume-to-project mapping",
			Warnings: []string{
				"raw volume deletion is not mapped to a compose project",
				"provide a compose-scoped command or explicit platform policy before execution",
			},
		}, nil
	case OpDockerSystemPrune:
		reason := "docker system prune is a host-global destructive operation and cannot be auto-executed without explicit platform policy"
		warnings := []string{
			"prune can delete resources across all projects on the host",
		}
		if containsFlag(plan.Flags, "--volumes") {
			reason += "; --volumes expands the blast radius to named volumes"
			warnings = append(warnings, "--volumes was requested")
		}
		if containsFlag(plan.Flags, "-a") || containsFlag(plan.Flags, "--all") {
			warnings = append(warnings, "all images may be pruned")
		}
		return Result{
			Allowed:       false,
			Operation:     plan.Operation,
			Scope:         "host-global",
			Flags:         append([]string(nil), plan.Flags...),
			ProtectionRun: false,
			Reason:        reason,
			Warnings:      warnings,
		}, nil
	default:
		return Result{
			Allowed:       false,
			Operation:     plan.Operation,
			Scope:         "unknown",
			ProtectionRun: false,
			Reason:        fmt.Sprintf("unsupported docker interception args for operation %q", plan.Operation),
		}, nil
	}
}

func containsFlag(flags []string, flag string) bool {
	for _, candidate := range flags {
		if candidate == flag {
			return true
		}
	}
	return false
}
