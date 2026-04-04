package kube

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGuardDeleteBlocksStatefulResources(t *testing.T) {
	temp := t.TempDir()
	path := filepath.Join(temp, "manifests.yaml")
	content := []byte("apiVersion: apps/v1\nkind: StatefulSet\nmetadata:\n  name: db\nspec:\n  template:\n    spec:\n      containers:\n        - name: db\n          image: postgres:16\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	result, err := GuardDelete(path)
	if err != nil {
		t.Fatalf("guard delete: %v", err)
	}
	if result.Allowed {
		t.Fatalf("expected blocked guard result, got %#v", result)
	}
}
