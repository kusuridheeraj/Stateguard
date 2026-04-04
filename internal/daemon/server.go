package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/kusuridheeraj/stateguard/internal/config"
	"github.com/kusuridheeraj/stateguard/internal/intercept"
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
	mux.HandleFunc("/readyz", s.handleReady)
	mux.HandleFunc("/api/v1/status", s.handleStatus)
	mux.HandleFunc("/api/v1/artifacts", s.handleArtifacts)
	mux.HandleFunc("/api/v1/adapters", s.handleAdapters)
	mux.HandleFunc("/api/v1/scheduler", s.handleScheduler)
	mux.HandleFunc("/api/v1/retention/preview", s.handleRetentionPreview)
	mux.HandleFunc("/api/v1/protect/compose", s.handleProtectCompose)
	mux.HandleFunc("/api/v1/restore/artifact", s.handleRestoreArtifact)
	mux.HandleFunc("/api/v1/guard/compose", s.handleGuardCompose)
	mux.HandleFunc("/api/v1/intercept/compose", s.handleInterceptCompose)
	mux.HandleFunc("/api/v1/guard/kube-delete", s.handleGuardKubeDelete)

	s.http = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Daemon.Host, cfg.Daemon.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return s, nil
}

func (s *Server) Handler() http.Handler {
	return s.http.Handler
}

func (s *Server) Run(ctx context.Context) error {
	s.control.RunStartupJobs(ctx)

	errCh := make(chan error, 1)

	go func() {
		s.logger.Info("daemon listening", "addr", s.http.Addr, "config_source", s.config.Source)
		if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.logger.Info("daemon shutting down")
		return s.http.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleReady(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready"))
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.control.Status("stateguard-daemon"))
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

func (s *Server) handleProtectCompose(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "missing path query parameter", http.StatusBadRequest)
		return
	}
	result, err := s.control.ProtectCompose(r.Context(), path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleRestoreArtifact(w http.ResponseWriter, r *http.Request) {
	artifactID := r.URL.Query().Get("id")
	if artifactID == "" {
		http.Error(w, "missing id query parameter", http.StatusBadRequest)
		return
	}
	result, err := s.control.RestoreArtifact(r.Context(), artifactID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleGuardCompose(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "missing path query parameter", http.StatusBadRequest)
		return
	}
	operation := intercept.Operation(r.URL.Query().Get("operation"))
	if operation == "" {
		operation = intercept.OpComposeDown
	}
	result, err := s.control.GuardComposeOperation(r.Context(), path, operation)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleInterceptCompose(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "missing path query parameter", http.StatusBadRequest)
		return
	}

	command := r.URL.Query().Get("command")
	execute := r.URL.Query().Get("execute") == "true"
	switch command {
	case "", "compose.down":
		withVolumes := r.URL.Query().Get("withVolumes") == "true"
		result, err := s.control.InterceptComposeDown(r.Context(), path, withVolumes, execute)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, result)
	case "compose.up":
		detached := r.URL.Query().Get("detached") != "false"
		build := r.URL.Query().Get("build") != "false"
		result, err := s.control.InterceptComposeUp(r.Context(), path, detached, build, execute)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, result)
	default:
		http.Error(w, "unsupported compose command", http.StatusBadRequest)
	}
}

func (s *Server) handleGuardKubeDelete(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "missing path query parameter", http.StatusBadRequest)
		return
	}
	result, err := s.control.GuardKubeDelete(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}
