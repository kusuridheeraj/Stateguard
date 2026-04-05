package kube

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kusuridheeraj/stateguard/pkg/types"
)

func TestReviewDeleteReturnsAdmissionReview(t *testing.T) {
	temp := t.TempDir()
	path := filepath.Join(temp, "manifests.yaml")
	content := []byte(`apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: demo
spec:
  template:
    spec:
      containers:
        - name: postgres
          image: postgres:16
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	review, err := ReviewDelete(path)
	if err != nil {
		t.Fatalf("review delete: %v", err)
	}
	if review.PolicyVersion != AdmissionPolicyVersion {
		t.Fatalf("expected policy version %q, got %q", AdmissionPolicyVersion, review.PolicyVersion)
	}
	if review.Decision.Allow {
		t.Fatalf("expected review to block stateful delete, got %#v", review)
	}
	if len(review.RequiredProtections) != 1 {
		t.Fatalf("expected 1 protection requirement, got %#v", review.RequiredProtections)
	}
}

func TestEnforceDeleteRequiresVerifiedProtection(t *testing.T) {
	temp := t.TempDir()
	path := filepath.Join(temp, "manifests.yaml")
	content := []byte(`apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: demo
spec:
  template:
    spec:
      containers:
        - name: postgres
          image: postgres:16
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	blocked, err := EnforceDelete(path, types.ProtectionState{})
	if err != nil {
		t.Fatalf("enforce delete without protection: %v", err)
	}
	if blocked.Decision.Allow {
		t.Fatalf("expected delete to be blocked without protection, got %#v", blocked)
	}

	allowed, err := EnforceDelete(path, types.ProtectionState{
		RecoveryPointExists: true,
		IntegrityValidated:  true,
		RestoreTested:       true,
	})
	if err != nil {
		t.Fatalf("enforce delete with protection: %v", err)
	}
	if !allowed.Decision.Allow {
		t.Fatalf("expected delete to be allowed with verified protection, got %#v", allowed)
	}
	if !allowed.ProtectionSatisfied {
		t.Fatalf("expected protection to be satisfied, got %#v", allowed)
	}
}
