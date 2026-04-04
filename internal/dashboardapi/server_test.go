package dashboardapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
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
