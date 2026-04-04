package compose

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
)

type ExecFunc func(context.Context, string, ...string) ([]byte, []byte, error)

type Runner struct {
	Exec ExecFunc
}

type RunResult struct {
	Command  []string `json:"command" yaml:"command"`
	Stdout   string   `json:"stdout" yaml:"stdout"`
	Stderr   string   `json:"stderr" yaml:"stderr"`
	ExitCode int      `json:"exitCode" yaml:"exitCode"`
}

func NewRunner() Runner {
	return Runner{Exec: defaultExec}
}

func (r Runner) Down(ctx context.Context, composePath string, removeOrphans bool, withVolumes bool) (RunResult, error) {
	args := []string{"compose", "-f", composePath, "down"}
	if withVolumes {
		args = append(args, "-v")
	}
	if removeOrphans {
		args = append(args, "--remove-orphans")
	}
	return r.run(ctx, "docker", args...)
}

func (r Runner) Up(ctx context.Context, composePath string, detached bool, build bool) (RunResult, error) {
	args := []string{"compose", "-f", composePath, "up"}
	if detached {
		args = append(args, "-d")
	}
	if build {
		args = append(args, "--build")
	}
	return r.run(ctx, "docker", args...)
}

func (r Runner) run(ctx context.Context, name string, args ...string) (RunResult, error) {
	execFn := r.Exec
	if execFn == nil {
		execFn = defaultExec
	}

	stdout, stderr, err := execFn(ctx, name, args...)
	result := RunResult{
		Command: append([]string{name}, args...),
		Stdout:  string(stdout),
		Stderr:  string(stderr),
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
	}
	return result, err
}

func defaultExec(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}
