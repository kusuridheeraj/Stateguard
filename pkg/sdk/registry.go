package sdk

import (
	"context"
	"sort"
)

type Registry struct {
	adapters []Adapter
}

func NewRegistry(adapters ...Adapter) *Registry {
	registry := &Registry{adapters: append([]Adapter{}, adapters...)}
	sort.SliceStable(registry.adapters, func(i, j int) bool {
		return registry.adapters[i].Metadata().Priority > registry.adapters[j].Metadata().Priority
	})
	return registry
}

func (r *Registry) Register(adapter Adapter) {
	r.adapters = append(r.adapters, adapter)
	sort.SliceStable(r.adapters, func(i, j int) bool {
		return r.adapters[i].Metadata().Priority > r.adapters[j].Metadata().Priority
	})
}

func (r *Registry) List() []MetadataView {
	out := make([]MetadataView, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		meta := adapter.Metadata()
		out = append(out, MetadataView{
			Name:         adapter.Name(),
			Official:     meta.Official,
			Priority:     meta.Priority,
			ImageHints:   append([]string{}, meta.ImageHints...),
			Description:  meta.Description,
			Capabilities: append([]string{}, meta.Capabilities...),
		})
	}
	return out
}

func (r *Registry) Resolve(ctx context.Context, target Target) (Adapter, DetectionResult, bool) {
	for _, adapter := range r.adapters {
		result, err := adapter.Detect(ctx, target)
		if err != nil {
			continue
		}
		if result.Supported {
			return adapter, result, true
		}
	}
	return nil, DetectionResult{}, false
}

func (r *Registry) GetByName(name string) (Adapter, bool) {
	for _, adapter := range r.adapters {
		if adapter.Name() == name {
			return adapter, true
		}
	}
	return nil, false
}

type MetadataView struct {
	Name         string   `json:"name" yaml:"name"`
	Official     bool     `json:"official" yaml:"official"`
	Priority     int      `json:"priority" yaml:"priority"`
	ImageHints   []string `json:"imageHints" yaml:"imageHints"`
	Description  string   `json:"description" yaml:"description"`
	Capabilities []string `json:"capabilities" yaml:"capabilities"`
}
