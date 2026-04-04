package backupexec

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kusuridheeraj/stateguard/pkg/sdk"
)

type CommandRunner interface {
	Run(context.Context, string, ...string) ([]byte, []byte, error)
}

type ExecFunc func(context.Context, string, ...string) ([]byte, []byte, error)

func (f ExecFunc) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	return f(ctx, name, args...)
}

type ComposeExecutor struct {
	Runner      CommandRunner
	ExecuteLive bool
}

type ExecutionResult struct {
	Mode         string   `json:"mode" yaml:"mode"`
	Files        []string `json:"files" yaml:"files"`
	CommandCount int      `json:"commandCount" yaml:"commandCount"`
	BytesWritten int64    `json:"bytesWritten" yaml:"bytesWritten"`
}

type executionMetadata struct {
	ServiceType string              `json:"serviceType" yaml:"serviceType"`
	Service     string              `json:"service" yaml:"service"`
	ComposeFile string              `json:"composeFile" yaml:"composeFile"`
	Mode        string              `json:"mode" yaml:"mode"`
	Supported   bool                `json:"supported" yaml:"supported"`
	Reason      string              `json:"reason,omitempty" yaml:"reason,omitempty"`
	Commands    []commandDefinition `json:"commands" yaml:"commands"`
}

type commandDefinition struct {
	FileName string   `json:"fileName" yaml:"fileName"`
	Command  []string `json:"command" yaml:"command"`
}

func NewComposeExecutor() ComposeExecutor {
	return ComposeExecutor{Runner: defaultRunner{}, ExecuteLive: false}
}

func (e ComposeExecutor) ExecuteProtection(ctx context.Context, composePath string, target sdk.Target, manifest map[string]any, bundleDir string) (ExecutionResult, error) {
	serviceType, _ := manifest["serviceType"].(string)
	commands, reason := buildCommands(serviceType, composePath, target.Name)

	metadata := executionMetadata{
		ServiceType: serviceType,
		Service:     target.Name,
		ComposeFile: composePath,
		Mode:        "planned",
		Supported:   len(commands) > 0,
		Reason:      reason,
		Commands:    commands,
	}
	if e.ExecuteLive {
		metadata.Mode = "executed"
	}

	planPath := filepath.Join(bundleDir, "execution.json")
	content, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("encode execution metadata: %w", err)
	}
	if err := os.WriteFile(planPath, content, 0o600); err != nil {
		return ExecutionResult{}, fmt.Errorf("write execution metadata: %w", err)
	}

	result := ExecutionResult{
		Mode:         metadata.Mode,
		Files:        []string{planPath},
		CommandCount: len(commands),
		BytesWritten: int64(len(content)),
	}
	if !e.ExecuteLive || len(commands) == 0 {
		return result, nil
	}

	if e.Runner == nil {
		return ExecutionResult{}, fmt.Errorf("live execution requested without command runner")
	}

	for _, command := range commands {
		stdout, stderr, err := e.Runner.Run(ctx, command.Command[0], command.Command[1:]...)
		outputPath := filepath.Join(bundleDir, command.FileName)
		payload := map[string]any{
			"command": command.Command,
			"stdout":  string(stdout),
			"stderr":  string(stderr),
		}
		if err != nil {
			payload["error"] = err.Error()
		}
		output, marshalErr := json.MarshalIndent(payload, "", "  ")
		if marshalErr != nil {
			return ExecutionResult{}, fmt.Errorf("encode command output: %w", marshalErr)
		}
		if writeErr := os.WriteFile(outputPath, output, 0o600); writeErr != nil {
			return ExecutionResult{}, fmt.Errorf("write command output: %w", writeErr)
		}
		result.Files = append(result.Files, outputPath)
		result.BytesWritten += int64(len(output))
		if err != nil {
			return result, fmt.Errorf("execute backup command for %s: %w", target.Name, err)
		}
	}

	return result, nil
}

type defaultRunner struct{}

func (defaultRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func buildCommands(serviceType string, composePath string, serviceName string) ([]commandDefinition, string) {
	switch serviceType {
	case "postgres":
		return []commandDefinition{
			{
				FileName: "backup.postgres.json",
				Command: []string{
					"docker", "compose", "-f", composePath, "exec", "-T", serviceName, "sh", "-lc",
					"pg_dumpall --clean --if-exists",
				},
			},
		}, ""
	case "redis":
		return []commandDefinition{
			{
				FileName: "backup.redis.json",
				Command: []string{
					"docker", "compose", "-f", composePath, "exec", "-T", serviceName, "sh", "-lc",
					"if [ -f /data/appendonly.aof ]; then ls -l /data/appendonly.aof; elif [ -f /data/dump.rdb ]; then ls -l /data/dump.rdb; else redis-cli SAVE >/dev/null 2>&1 && ls -l /data/dump.rdb; fi",
				},
			},
		}, ""
	case "vault":
		return []commandDefinition{
			{
				FileName: "backup.vault.json",
				Command: []string{
					"docker", "compose", "-f", composePath, "exec", "-T", serviceName, "sh", "-lc",
					"for path in /vault/file /vault/data; do if [ -d \"$path\" ]; then find \"$path\" -maxdepth 2 -type f | sort; fi; done",
				},
			},
		}, ""
	case "mysql":
		return []commandDefinition{
			{
				FileName: "backup.mysql.json",
				Command: []string{
					"docker", "compose", "-f", composePath, "exec", "-T", serviceName, "sh", "-lc",
					"mysqldump --all-databases --single-transaction --quick",
				},
			},
		}, ""
	case "mongodb":
		return []commandDefinition{
			{
				FileName: "backup.mongodb.json",
				Command: []string{
					"docker", "compose", "-f", composePath, "exec", "-T", serviceName, "sh", "-lc",
					"mongodump --archive --gzip",
				},
			},
		}, ""
	case "kafka":
		return []commandDefinition{
			{
				FileName: "backup.kafka.json",
				Command: []string{
					"docker", "compose", "-f", composePath, "exec", "-T", serviceName, "sh", "-lc",
					"for path in /var/lib/kafka /bitnami/kafka /var/lib/redpanda; do if [ -d \"$path\" ]; then find \"$path\" -maxdepth 2 -type f | sort; fi; done",
				},
			},
		}, ""
	default:
		return nil, fmt.Sprintf("no live compose backup execution plan for service type %q; bundle remains plan-only", serviceType)
	}
}
