package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/kusuridheeraj/stateguard/internal/artifacts"
	"github.com/kusuridheeraj/stateguard/internal/config"
	"github.com/kusuridheeraj/stateguard/internal/retention"
	"github.com/kusuridheeraj/stateguard/internal/scheduler"
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
	}

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
	return types.SystemStatus{
		ServiceName:     serviceName,
		Version:         c.build.Version,
		Mode:            c.config.Policy.Mode,
		ConfigSource:    c.config.Source,
		StartedAt:       c.startedAt,
		RuntimeTargets:  []types.RuntimeTarget{types.RuntimeCompose, types.RuntimeKubernetes},
		ProtectedScopes: 0,
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

func (c *ControlPlane) RunStartupJobs(ctx context.Context) {
	for _, name := range []string{"artifact-refresh", "retention-scan"} {
		if err := c.scheduler.RunOnce(ctx, name); err != nil {
			c.logger.Warn("startup job failed", "job", name, "error", err)
		}
	}
}
