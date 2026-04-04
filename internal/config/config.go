package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

var defaultCandidateFiles = []string{
	"safedata.yaml",
	"safedata.yml",
	"stateguard.yaml",
	"stateguard.yml",
}

type Config struct {
	Version string         `yaml:"version"`
	Project ProjectConfig  `yaml:"project"`
	Policy  PolicyConfig   `yaml:"policy"`
	Storage StorageConfig  `yaml:"storage"`
	Runtime RuntimeConfig  `yaml:"runtime"`
	Daemon  DaemonConfig   `yaml:"daemon"`
	API     APIConfig      `yaml:"api"`
	Source  string         `yaml:"-"`
	Raw     map[string]any `yaml:"-"`
}

type ProjectConfig struct {
	Name        string `yaml:"name"`
	Environment string `yaml:"environment"`
}

type PolicyConfig struct {
	Mode       string           `yaml:"mode"`
	Validation ValidationConfig `yaml:"validation"`
	Retention  RetentionConfig  `yaml:"retention"`
}

type ValidationConfig struct {
	Strategy       string            `yaml:"strategy"`
	IntegrityCheck bool              `yaml:"integrity_check"`
	AllowDegraded  bool              `yaml:"allow_degraded"`
	RestoreTest    RestoreTestConfig `yaml:"restore_test"`
}

type RestoreTestConfig struct {
	Enabled bool   `yaml:"enabled"`
	Cadence string `yaml:"cadence"`
}

type RetentionConfig struct {
	Window              string `yaml:"window"`
	MaxDiskUsagePercent int    `yaml:"max_disk_usage_percent"`
}

type StorageConfig struct {
	Local LocalStorageConfig `yaml:"local"`
}

type LocalStorageConfig struct {
	Path string `yaml:"path"`
}

type RuntimeConfig struct {
	Compose    ComposeRuntimeConfig    `yaml:"compose"`
	Kubernetes KubernetesRuntimeConfig `yaml:"kubernetes"`
}

type ComposeRuntimeConfig struct {
	ProjectBoundary string `yaml:"project_boundary"`
	LiveExecution   bool   `yaml:"live_execution"`
}

type KubernetesRuntimeConfig struct {
	Mode string `yaml:"mode"`
}

type DaemonConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type APIConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

func Default() Config {
	return Config{
		Version: "1",
		Project: ProjectConfig{
			Name:        "stateguard",
			Environment: "dev",
		},
		Policy: PolicyConfig{
			Mode: "fail-closed",
			Validation: ValidationConfig{
				Strategy:       "hybrid",
				IntegrityCheck: true,
				AllowDegraded:  false,
				RestoreTest: RestoreTestConfig{
					Enabled: true,
					Cadence: "periodic",
				},
			},
			Retention: RetentionConfig{
				Window:              "72h",
				MaxDiskUsagePercent: 20,
			},
		},
		Storage: StorageConfig{
			Local: LocalStorageConfig{
				Path: defaultArtifactPath(),
			},
		},
		Runtime: RuntimeConfig{
			Compose: ComposeRuntimeConfig{
				ProjectBoundary: "labels+compose_project",
				LiveExecution:   false,
			},
			Kubernetes: KubernetesRuntimeConfig{
				Mode: "beta",
			},
		},
		Daemon: DaemonConfig{
			Host: "127.0.0.1",
			Port: 7010,
		},
		API: APIConfig{
			Host: "127.0.0.1",
			Port: 7011,
		},
		Source: "defaults",
		Raw:    map[string]any{},
	}
}

func Load(path string) (Config, error) {
	if path == "" {
		discovered, err := discoverConfigPath()
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				cfg := Default()
				return cfg, nil
			}
			return Config{}, err
		}
		path = discovered
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	cfg := Default()
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	cfg.Source = path
	cfg.Raw = map[string]any{}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if c.Policy.Mode == "" {
		return errors.New("policy.mode is required")
	}
	if c.Policy.Mode != "fail-closed" && c.Policy.Mode != "warn" {
		return fmt.Errorf("unsupported policy.mode %q", c.Policy.Mode)
	}
	if c.Policy.Validation.Strategy == "" {
		return errors.New("policy.validation.strategy is required")
	}
	if _, err := time.ParseDuration(c.Policy.Retention.Window); err != nil {
		return fmt.Errorf("invalid policy.retention.window: %w", err)
	}
	if c.Policy.Retention.MaxDiskUsagePercent <= 0 || c.Policy.Retention.MaxDiskUsagePercent > 100 {
		return errors.New("policy.retention.max_disk_usage_percent must be between 1 and 100")
	}
	if c.Storage.Local.Path == "" {
		return errors.New("storage.local.path is required")
	}
	if c.Daemon.Port <= 0 || c.API.Port <= 0 {
		return errors.New("daemon.port and api.port must be positive")
	}
	return nil
}

func RenderDefaultYAML() ([]byte, error) {
	cfg := Default()
	return yaml.Marshal(cfg)
}

func discoverConfigPath() (string, error) {
	for _, candidate := range defaultCandidateFiles {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", os.ErrNotExist
}

func defaultArtifactPath() string {
	programData := os.Getenv("ProgramData")
	if programData != "" {
		return filepath.Join(programData, "Stateguard", "artifacts")
	}
	home, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(home, ".stateguard", "artifacts")
	}
	return ".artifacts"
}
