package dashboardapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/kusuridheeraj/stateguard/internal/config"
	"github.com/kusuridheeraj/stateguard/internal/service"
	"github.com/kusuridheeraj/stateguard/pkg/types"
)

type Server struct {
	logger    *slog.Logger
	config    config.Config
	build     types.BuildInfo
	startedAt time.Time
	control   *service.ControlPlane
	http      *http.Server
}

func NewServer(logger *slog.Logger, cfg config.Config, build types.BuildInfo) (*Server, error) {
	control, err := service.NewControlPlane(logger, cfg, build)
	if err != nil {
		return nil, err
	}

	s := &Server{
		logger:    logger,
		config:    cfg,
		build:     build,
		startedAt: time.Now().UTC(),
		control:   control,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/api/v1/status", s.handleStatus)
	mux.HandleFunc("/api/v1/overview", s.handleOverview)
	mux.HandleFunc("/api/v1/artifacts", s.handleArtifacts)
	mux.HandleFunc("/api/v1/adapters", s.handleAdapters)
	mux.HandleFunc("/api/v1/scheduler", s.handleScheduler)
	mux.HandleFunc("/api/v1/retention/preview", s.handleRetentionPreview)
	mux.Handle("/", s.staticHandler())

	s.http = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return s, nil
}

func (s *Server) Handler() http.Handler {
	return s.http.Handler
}

func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		s.logger.Info("dashboard api listening", "addr", s.http.Addr, "config_source", s.config.Source)
		if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.logger.Info("dashboard api shutting down")
		return s.http.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.control.Status("stateguard-dashboard-api"))
}

func (s *Server) handleOverview(w http.ResponseWriter, _ *http.Request) {
	status := s.control.Status("stateguard-dashboard-api")
	payload := map[string]any{
		"summary": map[string]any{
			"protectedScopes":    status.ProtectedScopes,
			"activeWarnings":     status.Artifacts.DegradedArtifacts,
			"blockedOperations":  0,
			"recoveryPointCount": status.Artifacts.Count,
		},
		"configSource": s.config.Source,
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleArtifacts(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": s.control.Artifacts()})
}

func (s *Server) handleAdapters(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": s.control.Adapters()})
}

func (s *Server) handleScheduler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"jobs": s.control.SchedulerJobs()})
}

func (s *Server) handleRetentionPreview(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.control.RetentionPreview())
}

func (s *Server) staticHandler() http.Handler {
	webDir := filepath.Join("web")
	filesystem := os.DirFS(webDir)
	fileServer := http.FileServer(http.FS(filesystem))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/" || r.URL.Path == "/index.html":
			http.ServeFile(w, r, filepath.Join(webDir, "index.html"))
			return
		case r.URL.Path == "/static/app.js":
			http.ServeFile(w, r, filepath.Join(webDir, "app.js"))
			return
		case r.URL.Path == "/static/styles.css":
			http.ServeFile(w, r, filepath.Join(webDir, "styles.css"))
			return
		case filepath.Ext(r.URL.Path) != "":
			if _, err := fs.Stat(filesystem, r.URL.Path[1:]); err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		http.ServeFile(w, r, filepath.Join(webDir, "index.html"))
	})
}

func writeJSON(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}
