package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func (s *Server) listSessions(ctx context.Context, filter SessionFilter) ([]SessionSummary, int, error) {
	whereClause, args := buildSessionWhere(filter)

	orderField := "s.created_at"
	if strings.TrimSpace(filter.Sort) == "uploaded_at" {
		orderField = "s.uploaded_at"
	}
	orderDir := "DESC"
	if strings.EqualFold(filter.Order, "asc") {
		orderDir = "ASC"
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	query := `
		SELECT
			s.id, s.tool, s.session_id, s.created_at, s.ended_at, s.cwd,
			s.uploaded_by, s.uploaded_at, s.duration_seconds, s.message_count, s.tool_use_count,
			first_msg.content,
			COALESCE(
				json_agg(json_build_object('key', t.tag_key, 'value', t.tag_value))
					FILTER (WHERE t.id IS NOT NULL),
				'[]'
			) AS tags
		FROM sessions s
		LEFT JOIN tags t ON t.session_id = s.id
		LEFT JOIN LATERAL (
			SELECT content
			FROM messages m
			WHERE m.session_id = s.id AND m.role = 'user'
			ORDER BY m.seq ASC
			LIMIT 1
		) first_msg ON true
		` + whereClause + `
		GROUP BY s.id, first_msg.content
		ORDER BY ` + orderField + ` ` + orderDir + `
		LIMIT $` + fmt.Sprintf("%d", len(args)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(args)+2) + `
	`
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var sessions []SessionSummary
	for rows.Next() {
		var summary SessionSummary
		var tagsRaw []byte
		var contentRaw []byte
		if err := rows.Scan(
			&summary.ID,
			&summary.Tool,
			&summary.SessionID,
			&summary.CreatedAt,
			&summary.EndedAt,
			&summary.Cwd,
			&summary.UploadedBy,
			&summary.UploadedAt,
			&summary.DurationSeconds,
			&summary.MessageCount,
			&summary.ToolUseCount,
			&contentRaw,
			&tagsRaw,
		); err != nil {
			return nil, 0, err
		}
		if len(tagsRaw) > 0 {
			_ = json.Unmarshal(tagsRaw, &summary.Tags)
		}
		summary.Summary = summarizeContent(contentRaw)
		sessions = append(sessions, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	total, err := s.countSessions(ctx, whereClause, args[:len(args)-2])
	if err != nil {
		return nil, 0, err
	}
	return sessions, total, nil
}

func (s *Server) countSessions(ctx context.Context, whereClause string, args []interface{}) (int, error) {
	query := `SELECT COUNT(*) FROM sessions s ` + whereClause
	var total int
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (s *Server) getSession(ctx context.Context, id string) (SessionDetail, error) {
	var detail SessionDetail
	row := s.db.QueryRowContext(ctx, `
		SELECT id, tool, session_id, created_at, ended_at, cwd, uploaded_by, uploaded_at,
			duration_seconds, message_count, tool_use_count
		FROM sessions
		WHERE id = $1
	`, id)
	if err := row.Scan(
		&detail.ID,
		&detail.Tool,
		&detail.SessionID,
		&detail.CreatedAt,
		&detail.EndedAt,
		&detail.Cwd,
		&detail.UploadedBy,
		&detail.UploadedAt,
		&detail.DurationSeconds,
		&detail.MessageCount,
		&detail.ToolUseCount,
	); err != nil {
		return SessionDetail{}, err
	}

	tags, err := s.listSessionTags(ctx, id)
	if err != nil {
		return SessionDetail{}, err
	}
	detail.Tags = tags

	messages, err := s.listMessages(ctx, id)
	if err != nil {
		return SessionDetail{}, err
	}
	detail.Messages = messages

	tools, err := s.listTools(ctx, id)
	if err != nil {
		return SessionDetail{}, err
	}
	detail.Tools = tools

	return detail, nil
}

func (s *Server) listSessionTags(ctx context.Context, sessionID string) ([]Tag, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT tag_key, tag_value
		FROM tags
		WHERE session_id = $1
		ORDER BY tag_key, tag_value
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.Key, &tag.Value); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tags, nil
}

func (s *Server) listMessages(ctx context.Context, sessionID string) ([]MessageDetail, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, timestamp, seq, role, model, content
		FROM messages
		WHERE session_id = $1
		ORDER BY seq
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []MessageDetail
	for rows.Next() {
		var msg MessageDetail
		if err := rows.Scan(&msg.ID, &msg.Timestamp, &msg.Seq, &msg.Role, &msg.Model, &msg.Content); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return messages, nil
}

func (s *Server) listTools(ctx context.Context, sessionID string) ([]ToolDetail, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, timestamp, tool_use_id, tool_name, input, output, is_error
		FROM tools
		WHERE session_id = $1
		ORDER BY timestamp
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tools []ToolDetail
	for rows.Next() {
		var tool ToolDetail
		if err := rows.Scan(&tool.ID, &tool.Timestamp, &tool.ToolUseID, &tool.ToolName, &tool.Input, &tool.Output, &tool.IsError); err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tools, nil
}

func (s *Server) listTags(ctx context.Context, key string, limit int) ([]TagCount, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `
		SELECT tag_key, tag_value, COUNT(*) AS count
		FROM tags
	`
	var args []interface{}
	if strings.TrimSpace(key) != "" {
		query += " WHERE tag_key = $1"
		args = append(args, key)
	}
	query += " GROUP BY tag_key, tag_value ORDER BY count DESC"
	if len(args) == 0 {
		query += " LIMIT $1"
		args = append(args, limit)
	} else {
		query += fmt.Sprintf(" LIMIT $%d", len(args)+1)
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []TagCount
	for rows.Next() {
		var tag TagCount
		if err := rows.Scan(&tag.Key, &tag.Value, &tag.Count); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tags, nil
}

func summarizeContent(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	type contentPart struct {
		Text    string `json:"text"`
		Content string `json:"content"`
	}
	var parts []contentPart
	if err := json.Unmarshal(raw, &parts); err == nil {
		for _, part := range parts {
			if strings.TrimSpace(part.Text) != "" {
				return trimSummary(part.Text)
			}
			if strings.TrimSpace(part.Content) != "" {
				return trimSummary(part.Content)
			}
		}
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return trimSummary(text)
	}
	return ""
}

func trimSummary(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 140 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= 140 {
		return value
	}
	return strings.TrimSpace(string(runes[:140])) + "..."
}

func buildSessionWhere(filter SessionFilter) (string, []interface{}) {
	var clauses []string
	var args []interface{}
	argIndex := 1

	clauses = append(clauses, "1=1")

	if strings.TrimSpace(filter.Tool) != "" {
		clauses = append(clauses, fmt.Sprintf("s.tool = $%d", argIndex))
		args = append(args, filter.Tool)
		argIndex++
	}

	if strings.TrimSpace(filter.UploadedBy) != "" {
		clauses = append(clauses, fmt.Sprintf("s.uploaded_by = $%d", argIndex))
		args = append(args, filter.UploadedBy)
		argIndex++
	}

	if strings.TrimSpace(filter.Query) != "" {
		like := "%" + filter.Query + "%"
		clauses = append(clauses, fmt.Sprintf(`(
			s.cwd ILIKE $%d
			OR EXISTS (
				SELECT 1 FROM messages m
				WHERE m.session_id = s.id AND m.content::text ILIKE $%d
			)
			OR EXISTS (
				SELECT 1 FROM tools tl
				WHERE tl.session_id = s.id AND (
					tl.tool_name ILIKE $%d OR tl.input::text ILIKE $%d OR tl.output::text ILIKE $%d
				)
			)
			OR EXISTS (
				SELECT 1 FROM tags tg
				WHERE tg.session_id = s.id AND (
					tg.tag_key ILIKE $%d OR tg.tag_value ILIKE $%d
				)
			)
		)`, argIndex, argIndex, argIndex, argIndex, argIndex, argIndex, argIndex))
		args = append(args, like)
		argIndex++
	}

	for i, tag := range filter.Tags {
		alias := fmt.Sprintf("t%d", i+1)
		clauses = append(clauses, fmt.Sprintf(
			`EXISTS (
				SELECT 1 FROM tags %s
				WHERE %s.session_id = s.id AND %s.tag_key = $%d AND %s.tag_value = $%d
			)`,
			alias, alias, alias, argIndex, alias, argIndex+1,
		))
		args = append(args, tag.Key, tag.Value)
		argIndex += 2
	}

	return "WHERE " + strings.Join(clauses, " AND "), args
}

func normalizeTagFilter(raw string) (Tag, error) {
	if strings.TrimSpace(raw) == "" {
		return Tag{}, errors.New("empty tag")
	}
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return Tag{}, errors.New("invalid tag")
	}
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	if key == "" || value == "" {
		return Tag{}, errors.New("invalid tag")
	}
	return Tag{Key: key, Value: value}, nil
}

func isNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
