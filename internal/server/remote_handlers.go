package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListSessions(w, r)
	case http.MethodPost:
		s.handleUploadSession(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	filter := SessionFilter{
		Tool:       strings.TrimSpace(r.URL.Query().Get("tool")),
		UploadedBy: strings.TrimSpace(r.URL.Query().Get("uploaded_by")),
		Query:      strings.TrimSpace(r.URL.Query().Get("q")),
	}

	for _, rawTag := range r.URL.Query()["tag"] {
		tag, err := normalizeTagFilter(rawTag)
		if err != nil {
			continue
		}
		filter.Tags = append(filter.Tags, tag)
	}

	sessions, err := s.listSessions(r.Context(), filter)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "server_error", "Failed to load sessions")
		return
	}

	resp := SessionsResponse{
		Sessions: sessions,
		Total:    len(sessions),
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSessionDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	rawID := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	if rawID == "" || rawID == "/" {
		s.writeError(w, http.StatusBadRequest, "invalid_request", "Missing session id")
		return
	}

	if _, err := uuid.Parse(rawID); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid session id")
		return
	}

	session, err := s.getSession(r.Context(), rawID)
	if err != nil {
		if isNotFound(err) {
			s.writeError(w, http.StatusNotFound, "session_not_found", "Session not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "server_error", "Failed to load session")
		return
	}

	resp := map[string]interface{}{
		"session": session,
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleTags(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			if parsed > 200 {
				parsed = 200
			}
			limit = parsed
		}
	}

	key := strings.TrimSpace(r.URL.Query().Get("key"))
	tags, err := s.listTags(r.Context(), key, limit)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "server_error", "Failed to load tags")
		return
	}

	resp := map[string]interface{}{
		"tags": tags,
	}
	s.writeJSON(w, http.StatusOK, resp)
}
