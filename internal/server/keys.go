package server

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (s *Server) handleKeys(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireJSONAuth(w, r)
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleListKeys(w, r, userID)
	case http.MethodPost:
		s.handleCreateKey(w, r, userID)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleKeyDetail(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireJSONAuth(w, r)
	if !ok {
		return
	}

	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/keys/")
	if id == "" || id == "/" {
		s.writeError(w, http.StatusBadRequest, "invalid_request", "Missing key id")
		return
	}
	if _, err := uuid.Parse(id); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid key id")
		return
	}

	result, err := s.db.ExecContext(r.Context(), `
		UPDATE api_keys
		SET is_active = false
		WHERE id = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "server_error", "Failed to revoke key")
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		s.writeError(w, http.StatusNotFound, "not_found", "API key not found")
		return
	}

	resp := map[string]string{
		"status":  "ok",
		"message": "API key revoked",
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleCreateKey(w http.ResponseWriter, r *http.Request, userID string) {
	var payload struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		s.writeError(w, http.StatusBadRequest, "invalid_request", "Name is required")
		return
	}

	keyValue, keyHash, keyPrefix, err := generateAPIKey()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "server_error", "Failed to generate key")
		return
	}

	var id string
	var createdAt string
	err = s.db.QueryRowContext(r.Context(), `
		INSERT INTO api_keys (key_hash, key_prefix, user_id, name)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`, keyHash, keyPrefix, userID, name).Scan(&id, &createdAt)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "server_error", "Failed to create key")
		return
	}

	resp := map[string]string{
		"id":         id,
		"key":        keyValue,
		"key_prefix": keyPrefix,
		"name":       name,
		"created_at": createdAt,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleListKeys(w http.ResponseWriter, r *http.Request, userID string) {
	rows, err := s.db.QueryContext(r.Context(), `
		SELECT id, key_prefix, name, created_at, last_used_at, is_active, usage_count
		FROM api_keys
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "server_error", "Failed to load keys")
		return
	}
	defer rows.Close()

	type keyRecord struct {
		ID         string  `json:"id"`
		KeyPrefix  string  `json:"key_prefix"`
		Name       string  `json:"name"`
		CreatedAt  string  `json:"created_at"`
		LastUsedAt *string `json:"last_used_at,omitempty"`
		IsActive   bool    `json:"is_active"`
		UsageCount int64   `json:"usage_count"`
	}
	keys := make([]keyRecord, 0)
	for rows.Next() {
		var rec keyRecord
		var lastUsedAt sql.NullTime
		if err := rows.Scan(&rec.ID, &rec.KeyPrefix, &rec.Name, &rec.CreatedAt, &lastUsedAt, &rec.IsActive, &rec.UsageCount); err != nil {
			s.writeError(w, http.StatusInternalServerError, "server_error", "Failed to load keys")
			return
		}
		if lastUsedAt.Valid {
			value := lastUsedAt.Time.Format(time.RFC3339Nano)
			rec.LastUsedAt = &value
		}
		keys = append(keys, rec)
	}
	if err := rows.Err(); err != nil {
		s.writeError(w, http.StatusInternalServerError, "server_error", "Failed to load keys")
		return
	}

	resp := map[string]interface{}{"keys": keys}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func generateAPIKey() (string, string, string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", "", "", err
	}
	suffix := hex.EncodeToString(buf)
	key := "tabs_" + suffix
	checksum := sha256.Sum256([]byte(key))
	hash := hex.EncodeToString(checksum[:])
	prefix := key
	if len(prefix) > 13 {
		prefix = prefix[:13]
	}
	return key, hash, prefix, nil
}
