package compose

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverComposeProject(t *testing.T) {
	temp := t.TempDir()
	path := filepath.Join(temp, "compose.yaml")
	content := []byte("name: demo\nservices:\n  db:\n    image: postgres:16\n    volumes:\n      - db-data:/var/lib/postgresql/data\n  api:\n    image: demo-api:latest\nvolumes:\n  db-data: {}\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write compose file: %v", err)
	}

	project, err := Discover(path)
	if err != nil {
		t.Fatalf("discover compose project: %v", err)
	}
	if project.Name != "demo" {
		t.Fatalf("expected project name demo, got %q", project.Name)
	}
	if len(project.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(project.Services))
	}
	if project.ScopeHint.StatefulServices != 1 {
		t.Fatalf("expected 1 stateful service, got %d", project.ScopeHint.StatefulServices)
	}
}
