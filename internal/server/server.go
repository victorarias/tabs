package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/victorarias/tabs/internal/logging"
)

type Server struct {
	db      *sql.DB
	baseURL string
	logger  *slog.Logger
	auth    Authenticator
}

func NewServer(db *sql.DB, baseURL string, logger *slog.Logger, auth Authenticator) *Server {
	if logger == nil {
		logger = logging.New("info", nil)
	}
	if auth == nil {
		auth = NoAuth{}
	}
	return &Server{
		db:      db,
		baseURL: baseURL,
		logger:  logger,
		auth:    auth,
	}
}

func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	httpServer := &http.Server{
		Handler:      s.routes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- httpServer.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/sessions", s.handleSessions)
	mux.HandleFunc("/api/sessions/", s.handleSessionDetail)
	mux.HandleFunc("/api/tags", s.handleTags)
	mux.HandleFunc("/api/keys", s.handleKeys)
	mux.HandleFunc("/api/keys/", s.handleKeyDetail)
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/app.js", s.handleStatic)
	mux.HandleFunc("/styles.css", s.handleStatic)
	mux.HandleFunc("/", s.handleRoot)
	return s.logRequests(mux)
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	if s.logger == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		s.logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if r.URL.Path != "/" && !strings.HasPrefix(r.URL.Path, "/sessions/") && r.URL.Path != "/search" && r.URL.Path != "/keys" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	s.serveUI(w, r, "index.html")
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.db.PingContext(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) writeError(w http.ResponseWriter, status int, code, message string) {
	resp := ErrorResponse{Error: ErrorPayload{Code: code, Message: message}}
	s.writeJSON(w, status, resp)
}

func (s *Server) requireJSONAuth(w http.ResponseWriter, r *http.Request) (string, bool) {
	if s.auth == nil {
		return "", true
	}
	user, err := s.auth.Authenticate(r)
	if err != nil {
		s.writeError(w, http.StatusForbidden, "forbidden", err.Error())
		return "", false
	}
	return user, true
}

func parsePort(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
