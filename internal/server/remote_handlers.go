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
	if _, ok := s.requireJSONAuth(w, r); !ok {
		return
	}
	filter := SessionFilter{
		Tool:       strings.TrimSpace(r.URL.Query().Get("tool")),
		UploadedBy: strings.TrimSpace(r.URL.Query().Get("uploaded_by")),
		Query:      strings.TrimSpace(r.URL.Query().Get("q")),
	}
	filter.Page = parsePositiveInt(r.URL.Query().Get("page"), 1)
	filter.Limit = parsePositiveInt(r.URL.Query().Get("limit"), 20)
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	filter.Sort = strings.TrimSpace(r.URL.Query().Get("sort"))
	filter.Order = strings.TrimSpace(r.URL.Query().Get("order"))

	for _, rawTag := range r.URL.Query()["tag"] {
		tag, err := normalizeTagFilter(rawTag)
		if err != nil {
			continue
		}
		filter.Tags = append(filter.Tags, tag)
	}

	sessions, total, err := s.listSessions(r.Context(), filter)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "server_error", "Failed to load sessions")
		return
	}

	resp := SessionsResponse{
		Sessions: sessions,
		Total:    total,
		Pagination: &Pagination{
			Page:       filter.Page,
			Limit:      filter.Limit,
			Total:      total,
			TotalPages: calcTotalPages(total, filter.Limit),
		},
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func parsePositiveInt(raw string, fallback int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func calcTotalPages(total, limit int) int {
	if limit <= 0 {
		return 1
	}
	pages := total / limit
	if total%limit != 0 {
		pages++
	}
	if pages == 0 {
		pages = 1
	}
	return pages
}

func (s *Server) handleSessionDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if _, ok := s.requireJSONAuth(w, r); !ok {
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
	if _, ok := s.requireJSONAuth(w, r); !ok {
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
