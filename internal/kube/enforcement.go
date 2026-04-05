package kube

import (
	"fmt"

	"github.com/kusuridheeraj/stateguard/pkg/types"
)

// AdmissionPolicyVersion is the current version of the Stateguard admission policy.
const AdmissionPolicyVersion = "admission.k8s.stateguard/v1"

// ProtectionRequirement describes a specific resource that requires protection before deletion.
type ProtectionRequirement struct {
	Kind      string `json:"kind" yaml:"kind"`
	Name      string `json:"name" yaml:"name"`
	Namespace string `json:"namespace" yaml:"namespace"`
	Reason    string `json:"reason" yaml:"reason"`
}

// AdmissionReview is a structured report evaluating the safety of a Kubernetes delete operation.
// It follows a pattern similar to Kubernetes admission review objects.
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

// Enforcer handles the logic for evaluating and enforcing state protection policies for Kubernetes.
type Enforcer struct {
	PolicyVersion string
}

// NewEnforcer creates a new Enforcer with the default policy version.
func NewEnforcer() Enforcer {
	return Enforcer{PolicyVersion: AdmissionPolicyVersion}
}

// ReviewDelete performs a static review of a manifest deletion without checking actual protection state.
func ReviewDelete(path string) (AdmissionReview, error) {
	descriptor, err := Discover(path)
	if err != nil {
		return AdmissionReview{}, err
	}
	return NewEnforcer().Review(descriptor), nil
}

// EnforceDelete performs a complete enforcement check for a manifest deletion against a verified protection state.
func EnforceDelete(path string, protection types.ProtectionState) (AdmissionReview, error) {
	descriptor, err := Discover(path)
	if err != nil {
		return AdmissionReview{}, err
	}
	return NewEnforcer().Enforce(descriptor, protection), nil
}

// Review evaluates the manifest descriptor and returns a review that blocks if stateful resources are found.
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

// Enforce evaluates the manifest descriptor against a provided protection state.
// It allows the operation only if all stateful resources have verified, non-degraded protection.
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
