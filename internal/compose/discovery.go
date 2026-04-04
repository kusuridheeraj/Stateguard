package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/kusuridheeraj/stateguard/pkg/types"
)

var knownStatefulImages = []string{
	"postgres",
	"redis",
	"mysql",
	"mongodb",
	"mongo",
	"kafka",
	"vault",
}

type File struct {
	Name     string                    `yaml:"name"`
	Services map[string]Service        `yaml:"services"`
	Volumes  map[string]map[string]any `yaml:"volumes"`
}

type Service struct {
	Image   string   `yaml:"image"`
	Volumes []string `yaml:"volumes"`
}

type ServiceDescriptor struct {
	Name               string   `json:"name" yaml:"name"`
	Image              string   `json:"image" yaml:"image"`
	HasPersistentMount bool     `json:"hasPersistentMount" yaml:"hasPersistentMount"`
	StatefulCandidate  bool     `json:"statefulCandidate" yaml:"statefulCandidate"`
	Reason             string   `json:"reason" yaml:"reason"`
	Mounts             []string `json:"mounts" yaml:"mounts"`
}

type ProjectDescriptor struct {
	Name      string               `json:"name" yaml:"name"`
	FilePath  string               `json:"filePath" yaml:"filePath"`
	Runtime   string               `json:"runtime" yaml:"runtime"`
	Detected  time.Time            `json:"detected" yaml:"detected"`
	Services  []ServiceDescriptor  `json:"services" yaml:"services"`
	ScopeHint types.ProtectedScope `json:"scopeHint" yaml:"scopeHint"`
}

func Discover(path string) (ProjectDescriptor, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return ProjectDescriptor{}, fmt.Errorf("read compose file: %w", err)
	}

	var file File
	if err := yaml.Unmarshal(content, &file); err != nil {
		return ProjectDescriptor{}, fmt.Errorf("parse compose file: %w", err)
	}

	projectName := file.Name
	if projectName == "" {
		projectName = filepath.Base(filepath.Dir(path))
	}

	descriptor := ProjectDescriptor{
		Name:     projectName,
		FilePath: path,
		Runtime:  "compose",
		Detected: time.Now().UTC(),
	}

	for serviceName, service := range file.Services {
		desc := ServiceDescriptor{
			Name:   serviceName,
			Image:  service.Image,
			Mounts: append([]string{}, service.Volumes...),
		}
		desc.HasPersistentMount = hasPersistentMount(service.Volumes)
		desc.StatefulCandidate, desc.Reason = detectStateful(service)
		descriptor.Services = append(descriptor.Services, desc)
		if desc.StatefulCandidate {
			descriptor.ScopeHint.StatefulServices++
		}
	}

	descriptor.ScopeHint = types.ProtectedScope{
		Name:             descriptor.Name,
		Runtime:          descriptor.Runtime,
		StatefulServices: descriptor.ScopeHint.StatefulServices,
		DetectedAt:       descriptor.Detected,
	}

	return descriptor, nil
}

func hasPersistentMount(volumes []string) bool {
	for _, volume := range volumes {
		if strings.Contains(volume, ":") {
			return true
		}
	}
	return false
}

func detectStateful(service Service) (bool, string) {
	image := strings.ToLower(service.Image)
	for _, candidate := range knownStatefulImages {
		if strings.Contains(image, candidate) {
			if hasPersistentMount(service.Volumes) {
				return true, fmt.Sprintf("known stateful image %q with persistent mount", candidate)
			}
			return true, fmt.Sprintf("known stateful image %q without declared persistent mount", candidate)
		}
	}
	if hasPersistentMount(service.Volumes) {
		return false, "persistent mounts declared but image is not in the stateful adapter set"
	}
	return false, "no stateful heuristic matched"
}
