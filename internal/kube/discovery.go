package kube

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type ResourceDescriptor struct {
	Kind              string   `json:"kind" yaml:"kind"`
	Name              string   `json:"name" yaml:"name"`
	Namespace         string   `json:"namespace" yaml:"namespace"`
	StatefulCandidate bool     `json:"statefulCandidate" yaml:"statefulCandidate"`
	Reason            string   `json:"reason" yaml:"reason"`
	Images            []string `json:"images" yaml:"images"`
}

type ManifestDescriptor struct {
	FilePath          string               `json:"filePath" yaml:"filePath"`
	Runtime           string               `json:"runtime" yaml:"runtime"`
	Detected          time.Time            `json:"detected" yaml:"detected"`
	Namespace         string               `json:"namespace" yaml:"namespace"`
	StatefulResources int                  `json:"statefulResources" yaml:"statefulResources"`
	Resources         []ResourceDescriptor `json:"resources" yaml:"resources"`
}

func Discover(path string) (ManifestDescriptor, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return ManifestDescriptor{}, fmt.Errorf("read kubernetes manifest: %w", err)
	}

	parts := strings.Split(string(content), "\n---")
	result := ManifestDescriptor{
		FilePath:  path,
		Runtime:   "kubernetes",
		Detected:  time.Now().UTC(),
		Resources: []ResourceDescriptor{},
	}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		var raw map[string]any
		if err := yaml.Unmarshal([]byte(part), &raw); err != nil {
			return ManifestDescriptor{}, fmt.Errorf("parse kubernetes manifest: %w", err)
		}

		resource := describeResource(raw)
		if preferredNamespace(resource) != "" && (result.Namespace == "" || result.Namespace == "default") {
			result.Namespace = preferredNamespace(resource)
		} else if resource.Namespace != "" && result.Namespace == "" {
			result.Namespace = resource.Namespace
		}
		if resource.StatefulCandidate {
			result.StatefulResources++
		}
		result.Resources = append(result.Resources, resource)
	}

	return result, nil
}

func preferredNamespace(resource ResourceDescriptor) string {
	if strings.EqualFold(resource.Kind, "Namespace") && resource.Name != "" {
		return resource.Name
	}
	if resource.Namespace != "" {
		return resource.Namespace
	}
	return ""
}

func describeResource(raw map[string]any) ResourceDescriptor {
	kind := stringValue(raw["kind"])
	metadata := nestedMap(raw, "metadata")
	spec := nestedMap(raw, "spec")

	descriptor := ResourceDescriptor{
		Kind:      kind,
		Name:      stringValue(metadata["name"]),
		Namespace: stringValue(metadata["namespace"]),
		Images:    collectImages(spec),
	}

	descriptor.StatefulCandidate, descriptor.Reason = classifyResource(kind, spec, descriptor.Images)
	if descriptor.Namespace == "" {
		descriptor.Namespace = "default"
	}

	return descriptor
}

func classifyResource(kind string, spec map[string]any, images []string) (bool, string) {
	switch strings.ToLower(kind) {
	case "statefulset":
		return true, "statefulset implies stable identity and persistent state"
	case "persistentvolumeclaim":
		return true, "persistent volume claim indicates durable workload storage"
	case "job", "cronjob":
		if containsStatefulImage(images) {
			return true, "batch workload uses known stateful image"
		}
	case "deployment":
		if hasVolumeMounts(spec) && containsStatefulImage(images) {
			return true, "deployment uses known stateful image with volumes"
		}
	}
	if containsStatefulImage(images) {
		return true, "known stateful image detected in workload"
	}
	return false, "no beta stateful heuristic matched"
}

func hasVolumeMounts(spec map[string]any) bool {
	template := nestedMap(spec, "template")
	podSpec := nestedMap(template, "spec")
	volumes, ok := podSpec["volumes"].([]any)
	return ok && len(volumes) > 0
}

func collectImages(spec map[string]any) []string {
	var images []string
	template := nestedMap(spec, "template")
	podSpec := nestedMap(template, "spec")

	for _, key := range []string{"containers", "initContainers"} {
		items, ok := podSpec[key].([]any)
		if !ok {
			continue
		}
		for _, item := range items {
			container, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if image := stringValue(container["image"]); image != "" {
				images = append(images, image)
			}
		}
	}

	if len(images) == 0 {
		if image := stringValue(spec["image"]); image != "" {
			images = append(images, image)
		}
	}
	return images
}

func containsStatefulImage(images []string) bool {
	for _, image := range images {
		image = strings.ToLower(image)
		for _, hint := range []string{"postgres", "redis", "mysql", "mongo", "mongodb", "kafka", "vault"} {
			if strings.Contains(image, hint) {
				return true
			}
		}
	}
	return false
}

func nestedMap(source map[string]any, key string) map[string]any {
	value, ok := source[key]
	if !ok {
		return map[string]any{}
	}
	mapped, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return mapped
}

func stringValue(value any) string {
	s, _ := value.(string)
	return s
}
