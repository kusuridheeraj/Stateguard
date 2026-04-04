package kube

import "fmt"

type GuardResult struct {
	Allowed           bool                 `json:"allowed" yaml:"allowed"`
	Reason            string               `json:"reason" yaml:"reason"`
	StatefulResources int                  `json:"statefulResources" yaml:"statefulResources"`
	Resources         []ResourceDescriptor `json:"resources" yaml:"resources"`
}

func GuardDelete(path string) (GuardResult, error) {
	descriptor, err := Discover(path)
	if err != nil {
		return GuardResult{}, err
	}

	allowed := descriptor.StatefulResources == 0
	reason := "no stateful Kubernetes resources detected"
	if !allowed {
		reason = fmt.Sprintf("stateful Kubernetes resources detected; beta enforcement requires explicit protection before delete")
	}

	return GuardResult{
		Allowed:           allowed,
		Reason:            reason,
		StatefulResources: descriptor.StatefulResources,
		Resources:         descriptor.Resources,
	}, nil
}
