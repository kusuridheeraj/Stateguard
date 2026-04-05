package dashboardapi

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

func TestStaticHandlerServesIndex(t *testing.T) {
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

	server, err := NewServer(logging.New(logging.Config{}), config.Default(), types.BuildInfo{Name: "stateguard"})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.staticHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); body == "" {
		t.Fatal("expected static index body")
	}
}

func TestOverviewEndpoint(t *testing.T) {
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

	server, err := NewServer(logging.New(logging.Config{}), config.Default(), types.BuildInfo{Name: "stateguard"})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/overview", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestDaemonStatusEndpoint(t *testing.T) {
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

	server, err := NewServer(logging.New(logging.Config{}), config.Default(), types.BuildInfo{Name: "stateguard"})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/daemon/status", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestDaemonProtectComposeEndpoint(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/api/v1/daemon/protect/compose?path=examples/windows-wsl2-compose/compose.yaml", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestDaemonGuardKubeDeleteEndpoint(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/api/v1/daemon/guard/kube-delete?path=examples/kubernetes-beta/manifests.yaml", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestDaemonInterceptComposeEndpoint(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/api/v1/daemon/intercept/compose?path=examples/windows-wsl2-compose/compose.yaml&command=compose.down", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestDaemonRestoreArtifactEndpoint(t *testing.T) {
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

	reqProtect := httptest.NewRequest(http.MethodGet, "/api/v1/daemon/protect/compose?path=examples/windows-wsl2-compose/compose.yaml", nil)
	recProtect := httptest.NewRecorder()
	server.Handler().ServeHTTP(recProtect, reqProtect)
	if recProtect.Code != http.StatusOK {
		t.Fatalf("expected protect 200, got %d body=%s", recProtect.Code, recProtect.Body.String())
	}

	artifacts := server.control.Artifacts()
	if len(artifacts) == 0 {
		t.Fatal("expected artifact after daemon protect")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/daemon/restore/artifact?id="+artifacts[0].ID, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestDaemonProtectKubeAndInterceptDockerEndpoints(t *testing.T) {
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

	reqProtect := httptest.NewRequest(http.MethodGet, "/api/v1/daemon/protect/kube?path=examples/kubernetes-beta/manifests.yaml", nil)
	recProtect := httptest.NewRecorder()
	server.Handler().ServeHTTP(recProtect, reqProtect)
	if recProtect.Code != http.StatusOK {
		t.Fatalf("expected kube protect 200, got %d body=%s", recProtect.Code, recProtect.Body.String())
	}

	reqIntercept := httptest.NewRequest(http.MethodGet, "/api/v1/daemon/intercept/docker?arg=compose&arg=-f&arg=examples/windows-wsl2-compose/compose.yaml&arg=down&arg=-v", nil)
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

	req := httptest.NewRequest(http.MethodGet, "/api/v1/daemon/enforce/kube-delete?path=examples/kubernetes-beta/manifests.yaml", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); !strings.Contains(body, `"review"`) {
		t.Fatalf("expected review payload in body, got %s", body)
	}
}

func TestDaemonInterceptDockerSystemPruneEndpointBlocksRawScope(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/api/v1/daemon/intercept/docker?arg=system&arg=prune&arg=--volumes", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "host-global") || !strings.Contains(rec.Body.String(), "prune") {
		t.Fatalf("expected host-global prune response, got %s", rec.Body.String())
	}
}
