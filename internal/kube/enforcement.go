package kube

import (
	"fmt"

	"github.com/kusuridheeraj/stateguard/pkg/types"
)

const AdmissionPolicyVersion = "admission.k8s.stateguard/v1"

type ProtectionRequirement struct {
	Kind      string `json:"kind" yaml:"kind"`
	Name      string `json:"name" yaml:"name"`
	Namespace string `json:"namespace" yaml:"namespace"`
	Reason    string `json:"reason" yaml:"reason"`
}

type AdmissionReview struct {
	Operation           string                  `json:"operation" yaml:"operation"`
	PolicyVersion       string                  `json:"policyVersion" yaml:"policyVersion"`
	Scope               string                  `json:"scope" yaml:"scope"`
	StatefulResources   int                     `json:"statefulResources" yaml:"statefulResources"`
	Resources           []ResourceDescriptor    `json:"resources" yaml:"resources"`
	RequiredProtections []ProtectionRequirement `json:"requiredProtections" yaml:"requiredProtections"`
	Protection          types.ProtectionState   `json:"protection" yaml:"protection"`
	Decision            types.PolicyDecision    `json:"decision" yaml:"decision"`
	ProtectionSatisfied bool                    `json:"protectionSatisfied" yaml:"protectionSatisfied"`
}

type Enforcer struct {
	PolicyVersion string
}

func NewEnforcer() Enforcer {
	return Enforcer{PolicyVersion: AdmissionPolicyVersion}
}

func ReviewDelete(path string) (AdmissionReview, error) {
	descriptor, err := Discover(path)
	if err != nil {
		return AdmissionReview{}, err
	}
	return NewEnforcer().Review(descriptor), nil
}

func EnforceDelete(path string, protection types.ProtectionState) (AdmissionReview, error) {
	descriptor, err := Discover(path)
	if err != nil {
		return AdmissionReview{}, err
	}
	return NewEnforcer().Enforce(descriptor, protection), nil
}

func (e Enforcer) Review(descriptor ManifestDescriptor) AdmissionReview {
	review := e.baseReview(descriptor)
	if review.StatefulResources == 0 {
		review.ProtectionSatisfied = true
		review.Decision = types.PolicyDecision{
			Allow:    true,
			Severity: "info",
			Reason:   "no stateful Kubernetes resources detected",
		}
		return review
	}

	review.Decision = types.PolicyDecision{
		Allow:    false,
		Severity: "block",
		Reason:   "stateful Kubernetes resources require verified protection before delete",
	}
	return review
}

func (e Enforcer) Enforce(descriptor ManifestDescriptor, protection types.ProtectionState) AdmissionReview {
	review := e.baseReview(descriptor)
	review.Protection = protection
	review.ProtectionSatisfied = protectionSatisfied(protection)

	if review.StatefulResources == 0 {
		review.Decision = types.PolicyDecision{
			Allow:    true,
			Severity: "info",
			Reason:   "no stateful Kubernetes resources detected",
		}
		return review
	}

	if review.ProtectionSatisfied {
		review.Decision = types.PolicyDecision{
			Allow:    true,
			Severity: "info",
			Reason:   "verified protection exists for all stateful Kubernetes resources",
		}
		return review
	}

	reason := "stateful Kubernetes resources require verified protection before delete"
	if protection.RecoveryPointExists && !protection.IntegrityValidated {
		reason = "recovery point exists, but integrity validation has not completed"
	} else if protection.IntegrityValidated && !protection.RestoreTested {
		reason = "recovery point is integrity-validated, but restore testing has not completed"
	} else if protection.Degraded {
		reason = "protection is degraded and cannot satisfy Kubernetes delete enforcement"
	}

	review.Decision = types.PolicyDecision{
		Allow:    false,
		Severity: "block",
		Reason:   reason,
	}
	return review
}

func (e Enforcer) baseReview(descriptor ManifestDescriptor) AdmissionReview {
	scope := descriptor.Namespace
	if scope == "" {
		scope = "default"
	}

	required := make([]ProtectionRequirement, 0, descriptor.StatefulResources)
	for _, resource := range descriptor.Resources {
		if !resource.StatefulCandidate {
			continue
		}
		required = append(required, ProtectionRequirement{
			Kind:      resource.Kind,
			Name:      resource.Name,
			Namespace: scope,
			Reason:    fmt.Sprintf("%s requires verified protection before delete", resource.Kind),
		})
	}

	return AdmissionReview{
		Operation:           "delete",
		PolicyVersion:       e.PolicyVersion,
		Scope:               scope,
		StatefulResources:   descriptor.StatefulResources,
		Resources:           descriptor.Resources,
		RequiredProtections: required,
	}
}

func protectionSatisfied(state types.ProtectionState) bool {
	return state.RecoveryPointExists && state.IntegrityValidated && state.RestoreTested && !state.Degraded
}
