package serve

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/sandbox"
	"github.com/vectra-guard/vectra-guard/internal/session"
)

// Server wraps the HTTP server for the local dashboard.
type Server struct {
	logger   *logging.Logger
	sessionM *session.Manager
	metrics  *sandbox.MetricsCollector
	mux      *http.ServeMux
}

// New creates a new dashboard server bound to the given workspace.
func New(workspace string, logger *logging.Logger, enableMetrics bool) (*Server, error) {
	mgr, err := session.NewManager(workspace, logger)
	if err != nil {
		return nil, fmt.Errorf("create session manager: %w", err)
	}
	metrics, err := sandbox.NewMetricsCollector("", enableMetrics)
	if err != nil {
		return nil, fmt.Errorf("create metrics collector: %w", err)
	}

	s := &Server{
		logger:   logger,
		sessionM: mgr,
		metrics:  metrics,
		mux:      http.NewServeMux(),
	}
	s.routes()
	return s, nil
}

// routes registers HTTP handlers.
func (s *Server) routes() {
	s.mux.HandleFunc("/api/sessions", s.handleSessions)
	s.mux.HandleFunc("/api/metrics", s.handleMetrics)
	s.mux.HandleFunc("/", s.handleIndex)
}

// ListenAndServe starts the HTTP server bound to 127.0.0.1:port.
func (s *Server) ListenAndServe(port int) error {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	s.logger.Info("serve dashboard started", map[string]any{
		"addr": addr,
	})
	srv := &http.Server{
		Handler:           s.mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return srv.Serve(ln)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<html><head><title>Vectra Guard Dashboard</title></head><body><h1>Vectra Guard Dashboard</h1><p>Use /api/sessions and /api/metrics for JSON endpoints.</p></body></html>`)
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessions, err := s.sessionM.List()
	if err != nil {
		s.logger.Error("list sessions failed", map[string]any{"error": err.Error()})
		http.Error(w, "failed to list sessions", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if err := enc.Encode(sessions); err != nil {
		s.logger.Error("encode sessions failed", map[string]any{"error": err.Error()})
	}
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if err := enc.Encode(s.metrics.GetMetrics()); err != nil {
		s.logger.Error("encode metrics failed", map[string]any{"error": err.Error()})
	}
}

// Helper for basic environment workspace detection (mainly used by CLI).
func DefaultWorkspace() (string, error) {
	if wd, err := os.Getwd(); err == nil {
		return wd, nil
	}
	return "", fmt.Errorf("unable to determine workspace")
}

