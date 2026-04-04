package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/kusuridheeraj/stateguard/pkg/types"
)

type JobFunc func(context.Context) error

type Scheduler struct {
	mu   sync.RWMutex
	jobs map[string]*job
}

type job struct {
	name          string
	cadence       time.Duration
	task          JobFunc
	enabled       bool
	lastRunAt     time.Time
	lastSuccessAt time.Time
	lastError     string
}

func New() *Scheduler {
	return &Scheduler{
		jobs: map[string]*job{},
	}
}

func (s *Scheduler) Register(name string, cadence time.Duration, task JobFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.jobs[name] = &job{
		name:    name,
		cadence: cadence,
		task:    task,
		enabled: true,
	}
}

func (s *Scheduler) Snapshot() []types.SchedulerJobStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]types.SchedulerJobStatus, 0, len(s.jobs))
	for _, job := range s.jobs {
		out = append(out, types.SchedulerJobStatus{
			Name:          job.name,
			Cadence:       job.cadence.String(),
			LastRunAt:     job.lastRunAt,
			LastSuccessAt: job.lastSuccessAt,
			LastError:     job.lastError,
			Enabled:       job.enabled,
		})
	}
	return out
}

func (s *Scheduler) RunOnce(ctx context.Context, name string) error {
	s.mu.RLock()
	job, ok := s.jobs[name]
	s.mu.RUnlock()
	if !ok {
		return nil
	}

	now := time.Now().UTC()
	err := job.task(ctx)

	s.mu.Lock()
	defer s.mu.Unlock()
	job.lastRunAt = now
	if err != nil {
		job.lastError = err.Error()
		return err
	}
	job.lastSuccessAt = now
	job.lastError = ""
	return nil
}
