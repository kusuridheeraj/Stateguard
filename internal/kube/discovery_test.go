package kube

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverManifest(t *testing.T) {
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
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
  namespace: demo
spec:
  template:
    spec:
      containers:
        - name: api
          image: ghcr.io/example/api:latest
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	descriptor, err := Discover(path)
	if err != nil {
		t.Fatalf("discover manifest: %v", err)
	}
	if descriptor.Namespace != "demo" {
		t.Fatalf("expected namespace demo, got %q", descriptor.Namespace)
	}
	if descriptor.StatefulResources != 1 {
		t.Fatalf("expected 1 stateful resource, got %d", descriptor.StatefulResources)
	}
	if len(descriptor.Resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(descriptor.Resources))
	}
}
