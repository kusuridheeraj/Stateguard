package kube

type GuardResult struct {
	Allowed           bool                 `json:"allowed" yaml:"allowed"`
	Reason            string               `json:"reason" yaml:"reason"`
	StatefulResources int                  `json:"statefulResources" yaml:"statefulResources"`
	Resources         []ResourceDescriptor `json:"resources" yaml:"resources"`
}

func GuardDelete(path string) (GuardResult, error) {
	review, err := ReviewDelete(path)
	if err != nil {
		return GuardResult{}, err
	}

	return GuardResult{
		Allowed:           review.Decision.Allow,
		Reason:            review.Decision.Reason,
		StatefulResources: review.StatefulResources,
		Resources:         review.Resources,
	}, nil
}
