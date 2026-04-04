package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/kusuridheeraj/stateguard/adapters/generic"
	"github.com/kusuridheeraj/stateguard/adapters/kafka"
	"github.com/kusuridheeraj/stateguard/adapters/mongodb"
	"github.com/kusuridheeraj/stateguard/adapters/mysql"
	"github.com/kusuridheeraj/stateguard/adapters/postgres"
	"github.com/kusuridheeraj/stateguard/adapters/redis"
	"github.com/kusuridheeraj/stateguard/adapters/vault"
	"github.com/kusuridheeraj/stateguard/internal/artifacts"
	"github.com/kusuridheeraj/stateguard/internal/compose"
	"github.com/kusuridheeraj/stateguard/internal/config"
	"github.com/kusuridheeraj/stateguard/internal/intercept"
	"github.com/kusuridheeraj/stateguard/internal/kube"
	"github.com/kusuridheeraj/stateguard/internal/orchestrator"
	"github.com/kusuridheeraj/stateguard/internal/retention"
	"github.com/kusuridheeraj/stateguard/internal/scheduler"
	"github.com/kusuridheeraj/stateguard/pkg/sdk"
	"github.com/kusuridheeraj/stateguard/pkg/types"
)

type ControlPlane struct {
	logger          *slog.Logger
	config          config.Config
	build           types.BuildInfo
	startedAt       time.Time
	artifacts       *artifacts.Store
	scheduler       *scheduler.Scheduler
	retentionEngine retention.Engine
	registry        *sdk.Registry
	protector       *orchestrator.Protector
	interceptor     intercept.Evaluator
	composeRunner   compose.Runner
}

type ComposeInterception struct {
	Guard     intercept.Result   `json:"guard" yaml:"guard"`
	Executed  bool               `json:"executed" yaml:"executed"`
	Command   string             `json:"command" yaml:"command"`
	Compose   string             `json:"compose" yaml:"compose"`
	RunResult *compose.RunResult `json:"runResult,omitempty" yaml:"runResult,omitempty"`
}

func NewControlPlane(logger *slog.Logger, cfg config.Config, build types.BuildInfo) (*ControlPlane, error) {
	store, err := artifacts.NewStore(cfg.Storage.Local.Path)
	if err != nil {
		return nil, err
	}

	window, err := time.ParseDuration(cfg.Policy.Retention.Window)
	if err != nil {
		return nil, err
	}

	cp := &ControlPlane{
		logger:          logger,
		config:          cfg,
		build:           build,
		startedAt:       time.Now().UTC(),
		artifacts:       store,
		scheduler:       scheduler.New(),
		retentionEngine: retention.NewEngine(window),
		registry: sdk.NewRegistry(
			postgres.New(),
			redis.New(),
			vault.New(),
			mysql.New(),
			mongodb.New(),
			kafka.New(),
			generic.New(),
		),
	}
	cp.protector = orchestrator.NewProtector(cp.artifacts, cp.registry)
	cp.interceptor = intercept.Evaluator{
		Mode:           cfg.Policy.Mode,
		ProtectCompose: cp.protector.ProtectCompose,
	}
	cp.composeRunner = compose.NewRunner()

	cp.registerJobs()
	return cp, nil
}

func (c *ControlPlane) registerJobs() {
	c.scheduler.Register("artifact-refresh", 5*time.Minute, func(context.Context) error {
		c.logger.Info("artifact refresh job executed")
		return nil
	})
	c.scheduler.Register("retention-scan", 15*time.Minute, func(context.Context) error {
		c.logger.Info("retention scan job executed")
		return nil
	})
}

func (c *ControlPlane) Status(serviceName string) types.SystemStatus {
	records := c.artifacts.List()
	scopes := make(map[string]struct{})
	for _, record := range records {
		scopes[record.Scope] = struct{}{}
	}

	return types.SystemStatus{
		ServiceName:     serviceName,
		Version:         c.build.Version,
		Mode:            c.config.Policy.Mode,
		ConfigSource:    c.config.Source,
		StartedAt:       c.startedAt,
		RuntimeTargets:  []types.RuntimeTarget{types.RuntimeCompose, types.RuntimeKubernetes},
		ProtectedScopes: len(scopes),
		Artifacts:       c.artifacts.Summary(),
	}
}

func (c *ControlPlane) Artifacts() []types.ArtifactRecord {
	return c.artifacts.List()
}

func (c *ControlPlane) SchedulerJobs() []types.SchedulerJobStatus {
	return c.scheduler.Snapshot()
}

func (c *ControlPlane) RetentionPreview() retention.Plan {
	records := c.artifacts.List()
	return c.retentionEngine.Evaluate(records, retention.Snapshot{}, c.config.Policy.Retention.MaxDiskUsagePercent, time.Now().UTC())
}

func (c *ControlPlane) Adapters() []sdk.MetadataView {
	return c.registry.List()
}

func (c *ControlPlane) ProtectCompose(ctx context.Context, path string) (orchestrator.ProtectReport, error) {
	return c.protector.ProtectCompose(ctx, path)
}

func (c *ControlPlane) GuardComposeOperation(ctx context.Context, path string, operation intercept.Operation) (intercept.Result, error) {
	return c.interceptor.EvaluateComposeOperation(ctx, path, operation)
}

func (c *ControlPlane) InterceptComposeDown(ctx context.Context, path string, withVolumes bool, execute bool) (ComposeInterception, error) {
	operation := intercept.OpComposeDown
	if withVolumes {
		operation = intercept.OpComposeDownWithVolumes
	}

	guard, err := c.GuardComposeOperation(ctx, path, operation)
	result := ComposeInterception{
		Guard:    guard,
		Command:  string(operation),
		Compose:  path,
		Executed: false,
	}
	if err != nil {
		return result, err
	}
	if !execute || !guard.Allowed {
		return result, nil
	}

	runResult, err := c.composeRunner.Down(ctx, path, true, withVolumes)
	result.Executed = true
	result.RunResult = &runResult
	return result, err
}

func (c *ControlPlane) InterceptComposeUp(ctx context.Context, path string, detached bool, build bool, execute bool) (ComposeInterception, error) {
	result := ComposeInterception{
		Command:  "compose.up",
		Compose:  path,
		Executed: false,
	}
	if !execute {
		return result, nil
	}

	runResult, err := c.composeRunner.Up(ctx, path, detached, build)
	result.Executed = true
	result.RunResult = &runResult
	return result, err
}

func (c *ControlPlane) GuardKubeDelete(path string) (kube.GuardResult, error) {
	return kube.GuardDelete(path)
}

func (c *ControlPlane) RunStartupJobs(ctx context.Context) {
	for _, name := range []string{"artifact-refresh", "retention-scan"} {
		if err := c.scheduler.RunOnce(ctx, name); err != nil {
			c.logger.Warn("startup job failed", "job", name, "error", err)
		}
	}
}
