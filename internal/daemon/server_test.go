package daemon

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kusuridheeraj/stateguard/internal/config"
	"github.com/kusuridheeraj/stateguard/pkg/logging"
	"github.com/kusuridheeraj/stateguard/pkg/types"
)

func TestStatusEndpoint(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.Local.Path = filepath.Join(t.TempDir(), "artifacts")

	server, err := NewServer(logging.New(logging.Config{}), cfg, types.BuildInfo{Name: "stateguard"})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestGuardComposeEndpoint(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.Local.Path = filepath.Join(t.TempDir(), "artifacts")

	server, err := NewServer(logging.New(logging.Config{}), cfg, types.BuildInfo{Name: "stateguard"})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	path := filepath.Clean(filepath.Join("..", "..", "examples", "windows-wsl2-compose", "compose.yaml"))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/guard/compose?path="+path, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestProtectComposeEndpoint(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.Local.Path = filepath.Join(t.TempDir(), "artifacts")

	server, err := NewServer(logging.New(logging.Config{}), cfg, types.BuildInfo{Name: "stateguard"})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	path := filepath.Clean(filepath.Join("..", "..", "examples", "windows-wsl2-compose", "compose.yaml"))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/protect/compose?path="+path, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGuardKubeDeleteEndpoint(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.Local.Path = filepath.Join(t.TempDir(), "artifacts")

	server, err := NewServer(logging.New(logging.Config{}), cfg, types.BuildInfo{Name: "stateguard"})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	path := filepath.Clean(filepath.Join("..", "..", "examples", "kubernetes-beta", "manifests.yaml"))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/guard/kube-delete?path="+path, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestInterceptComposeEndpointWithoutExecution(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.Local.Path = filepath.Join(t.TempDir(), "artifacts")

	server, err := NewServer(logging.New(logging.Config{}), cfg, types.BuildInfo{Name: "stateguard"})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	path := filepath.Clean(filepath.Join("..", "..", "examples", "windows-wsl2-compose", "compose.yaml"))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/intercept/compose?path="+path+"&command=compose.down", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRestoreArtifactEndpoint(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.Local.Path = filepath.Join(t.TempDir(), "artifacts")

	server, err := NewServer(logging.New(logging.Config{}), cfg, types.BuildInfo{Name: "stateguard"})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	path := filepath.Clean(filepath.Join("..", "..", "examples", "windows-wsl2-compose", "compose.yaml"))
	reqProtect := httptest.NewRequest(http.MethodGet, "/api/v1/protect/compose?path="+path, nil)
	recProtect := httptest.NewRecorder()
	server.Handler().ServeHTTP(recProtect, reqProtect)
	if recProtect.Code != http.StatusOK {
		t.Fatalf("expected protect 200, got %d body=%s", recProtect.Code, recProtect.Body.String())
	}

	artifacts := server.control.Artifacts()
	if len(artifacts) == 0 {
		t.Fatal("expected artifact after protect")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/restore/artifact?id="+artifacts[0].ID, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestProtectKubeAndInterceptDockerEndpoints(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.Local.Path = filepath.Join(t.TempDir(), "artifacts")

	server, err := NewServer(logging.New(logging.Config{}), cfg, types.BuildInfo{Name: "stateguard"})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	kubePath := filepath.Clean(filepath.Join("..", "..", "examples", "kubernetes-beta", "manifests.yaml"))
	reqProtect := httptest.NewRequest(http.MethodGet, "/api/v1/protect/kube?path="+kubePath, nil)
	recProtect := httptest.NewRecorder()
	server.Handler().ServeHTTP(recProtect, reqProtect)
	if recProtect.Code != http.StatusOK {
		t.Fatalf("expected kube protect 200, got %d body=%s", recProtect.Code, recProtect.Body.String())
	}

	composePath := filepath.Clean(filepath.Join("..", "..", "examples", "windows-wsl2-compose", "compose.yaml"))
	reqIntercept := httptest.NewRequest(http.MethodGet, "/api/v1/intercept/docker?arg=compose&arg=-f&arg="+composePath+"&arg=down&arg=-v", nil)
	recIntercept := httptest.NewRecorder()
	server.Handler().ServeHTTP(recIntercept, reqIntercept)
	if recIntercept.Code != http.StatusOK {
		t.Fatalf("expected docker intercept 200, got %d body=%s", recIntercept.Code, recIntercept.Body.String())
	}
}

func TestDaemonEnforceKubeDeleteEndpointReturnsAdmissionReview(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to resolve current file path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))

	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("chdir repo root: %v", err)
	}
	defer func() { _ = os.Chdir(previous) }()

	cfg := config.Default()
	cfg.Storage.Local.Path = filepath.Join(t.TempDir(), "artifacts")

	server, err := NewServer(logging.New(logging.Config{}), cfg, types.BuildInfo{Name: "stateguard"})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/enforce/kube-delete?path=examples/kubernetes-beta/manifests.yaml", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); !strings.Contains(body, `"review"`) {
		t.Fatalf("expected review payload in body, got %s", body)
	}
}

func TestDaemonInterceptDockerVolumeRemoveEndpointBlocksRawScope(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.Local.Path = filepath.Join(t.TempDir(), "artifacts")

	server, err := NewServer(logging.New(logging.Config{}), cfg, types.BuildInfo{Name: "stateguard"})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/intercept/docker?arg=volume&arg=rm&arg=-f&arg=cache-a", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "host-global") || !strings.Contains(rec.Body.String(), "cache-a") {
		t.Fatalf("expected host-global blocked response, got %s", rec.Body.String())
	}
}
