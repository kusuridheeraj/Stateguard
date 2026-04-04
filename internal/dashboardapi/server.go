package dashboardapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/kusuridheeraj/stateguard/internal/config"
	"github.com/kusuridheeraj/stateguard/pkg/types"
)

type Server struct {
	logger    *slog.Logger
	config    config.Config
	build     types.BuildInfo
	startedAt time.Time
	http      *http.Server
}

func NewServer(logger *slog.Logger, cfg config.Config, build types.BuildInfo) *Server {
	s := &Server{
		logger:    logger,
		config:    cfg,
		build:     build,
		startedAt: time.Now().UTC(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/api/v1/status", s.handleStatus)
	mux.HandleFunc("/api/v1/overview", s.handleOverview)

	s.http = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return s
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
	writeJSON(w, http.StatusOK, types.SystemStatus{
		ServiceName:     "stateguard-dashboard-api",
		Version:         s.build.Version,
		Mode:            s.config.Policy.Mode,
		ConfigSource:    s.config.Source,
		StartedAt:       s.startedAt,
		RuntimeTargets:  []types.RuntimeTarget{types.RuntimeCompose, types.RuntimeKubernetes},
		ProtectedScopes: 0,
	})
}

func (s *Server) handleOverview(w http.ResponseWriter, _ *http.Request) {
	payload := map[string]any{
		"summary": map[string]any{
			"protectedScopes":    0,
			"activeWarnings":     0,
			"blockedOperations":  0,
			"recoveryPointCount": 0,
		},
		"configSource": s.config.Source,
	}
	writeJSON(w, http.StatusOK, payload)
}

func writeJSON(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}
