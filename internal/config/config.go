package config

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Local      LocalConfig
	Remote     RemoteConfig
	Cursor     CursorConfig
	ClaudeCode ClaudeCodeConfig
}

type LocalConfig struct {
	UIPort                     int
	LogLevel                   string
	EmptySessionRetentionHours int // 0 = keep forever, >0 = delete empty sessions older than N hours
}

type RemoteConfig struct {
	ServerURL   string
	APIKey      string
	AutoPush    bool
	DefaultTags []string
}

type CursorConfig struct {
	DBPath       string
	PollInterval int
}

type ClaudeCodeConfig struct {
	ProjectsDir string
}

func Default() Config {
	return Config{
		Local: LocalConfig{
			UIPort:                     3787,
			LogLevel:                   "info",
			EmptySessionRetentionHours: 24, // Delete empty sessions after 24 hours by default
		},
		Remote: RemoteConfig{
			ServerURL:   "https://tabs.company.com",
			APIKey:      "",
			AutoPush:    false,
			DefaultTags: []string{},
		},
		Cursor: CursorConfig{
			DBPath:       "",
			PollInterval: 2,
		},
		ClaudeCode: ClaudeCodeConfig{
			ProjectsDir: "",
		},
	}
}

func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".tabs", "config.toml"), nil
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	return Parse(data)
}

func Parse(data []byte) (Config, error) {
	cfg := Default()
	var section string

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := stripTomlComment(strings.TrimSpace(scanner.Text()))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if err := ApplyValue(&cfg, section, key, value); err != nil {
			return Config{}, err
		}
	}
	if err := scanner.Err(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func stripTomlComment(line string) string {
	if !strings.Contains(line, "#") {
		return line
	}
	var b strings.Builder
	inQuotes := false
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if ch == '"' {
			inQuotes = !inQuotes
		}
		if ch == '#' && !inQuotes {
			break
		}
		b.WriteByte(ch)
	}
	return strings.TrimSpace(b.String())
}

func ApplyValue(cfg *Config, section, key, raw string) error {
	value, err := parseTomlValue(raw)
	if err != nil {
		return err
	}

	switch section {
	case "local":
		switch key {
		case "ui_port":
			port, err := toInt(value)
			if err != nil {
				return err
			}
			cfg.Local.UIPort = port
		case "log_level":
			text, err := toString(value)
			if err != nil {
				return err
			}
			cfg.Local.LogLevel = text
		case "empty_session_retention_hours":
			hours, err := toInt(value)
			if err != nil {
				return err
			}
			cfg.Local.EmptySessionRetentionHours = hours
		}
	case "remote":
		switch key {
		case "server_url":
			text, err := toString(value)
			if err != nil {
				return err
			}
			cfg.Remote.ServerURL = text
		case "api_key":
			text, err := toString(value)
			if err != nil {
				return err
			}
			cfg.Remote.APIKey = text
		case "auto_push":
			b, err := toBool(value)
			if err != nil {
				return err
			}
			cfg.Remote.AutoPush = b
		case "default_tags":
			arr, err := toStringSlice(value)
			if err != nil {
				return err
			}
			cfg.Remote.DefaultTags = arr
		}
	case "cursor":
		switch key {
		case "db_path":
			text, err := toString(value)
			if err != nil {
				return err
			}
			cfg.Cursor.DBPath = text
		case "poll_interval":
			interval, err := toInt(value)
			if err != nil {
				return err
			}
			cfg.Cursor.PollInterval = interval
		}
	case "claude_code":
		switch key {
		case "projects_dir":
			text, err := toString(value)
			if err != nil {
				return err
			}
			cfg.ClaudeCode.ProjectsDir = text
		}
	}

	return nil
}

func parseTomlValue(raw string) (interface{}, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
		inner := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
		if inner == "" {
			return []string{}, nil
		}
		parts := splitComma(inner)
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			unquoted, err := strconv.Unquote(part)
			if err != nil {
				unquoted = part
			}
			values = append(values, unquoted)
		}
		return values, nil
	}
	if strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"") {
		unquoted, err := strconv.Unquote(trimmed)
		if err != nil {
			return nil, err
		}
		return unquoted, nil
	}
	if trimmed == "true" || trimmed == "false" {
		return trimmed == "true", nil
	}
	if n, err := strconv.Atoi(trimmed); err == nil {
		return n, nil
	}
	return trimmed, nil
}

func splitComma(input string) []string {
	parts := strings.Split(input, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	return parts
}

func toString(value interface{}) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case fmt.Stringer:
		return v.String(), nil
	default:
		return "", fmt.Errorf("expected string, got %T", value)
	}
}

func toInt(value interface{}) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("expected int, got %T", value)
	}
}

func toBool(value interface{}) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	default:
		return false, fmt.Errorf("expected bool, got %T", value)
	}
}

func toStringSlice(value interface{}) ([]string, error) {
	switch v := value.(type) {
	case []string:
		return v, nil
	case string:
		if v == "" {
			return []string{}, nil
		}
		return splitComma(v), nil
	default:
		return nil, fmt.Errorf("expected string slice, got %T", value)
	}
}

func ApplySet(cfg *Config, key, rawValue string) error {
	normalized := normalizeKey(key)

	switch normalized {
	case "remote.server_url", "server.url", "server_url":
		value := strings.TrimSpace(rawValue)
		if value == "" {
			cfg.Remote.ServerURL = ""
			return nil
		}
		if !strings.HasPrefix(value, "https://") {
			return errors.New("server_url must start with https://")
		}
		cfg.Remote.ServerURL = value
		return nil
	case "remote.api_key", "api.key", "api_key", "api-key":
		value := strings.TrimSpace(rawValue)
		if value != "" && (!strings.HasPrefix(value, "tabs_") || len(value) < 36) {
			return errors.New("api_key must start with tabs_ and be at least 36 characters")
		}
		cfg.Remote.APIKey = value
		return nil
	case "remote.auto_push", "auto.push", "auto_push", "auto-push":
		b, err := strconv.ParseBool(rawValue)
		if err != nil {
			return errors.New("auto_push must be true or false")
		}
		cfg.Remote.AutoPush = b
		return nil
	case "remote.default_tags", "default.tags", "default_tags", "default-tags":
		cfg.Remote.DefaultTags = parseTags(rawValue)
		return nil
	case "local.ui_port", "ui.port", "ui_port", "ui-port":
		port, err := strconv.Atoi(rawValue)
		if err != nil {
			return errors.New("ui_port must be a number")
		}
		if port < 1024 || port > 65535 {
			return errors.New("ui_port must be between 1024 and 65535")
		}
		cfg.Local.UIPort = port
		return nil
	case "local.log_level", "log.level", "log_level", "log-level":
		level := strings.ToLower(strings.TrimSpace(rawValue))
		if level == "" {
			return errors.New("log_level cannot be empty")
		}
		switch level {
		case "debug", "info", "warn", "error":
			cfg.Local.LogLevel = level
			return nil
		default:
			return errors.New("log_level must be one of: debug, info, warn, error")
		}
	case "local.empty_session_retention_hours", "empty_session_retention_hours", "empty-session-retention-hours":
		hours, err := strconv.Atoi(rawValue)
		if err != nil {
			return errors.New("empty_session_retention_hours must be a number")
		}
		if hours < 0 {
			return errors.New("empty_session_retention_hours must be >= 0")
		}
		cfg.Local.EmptySessionRetentionHours = hours
		return nil
	case "cursor.db_path", "cursor.db-path", "db.path", "db_path", "db-path":
		path := ExpandHome(strings.TrimSpace(rawValue))
		cfg.Cursor.DBPath = path
		return nil
	case "cursor.poll_interval", "cursor.poll-interval", "poll.interval", "poll_interval", "poll-interval":
		interval, err := strconv.Atoi(rawValue)
		if err != nil {
			return errors.New("poll_interval must be a number")
		}
		if interval < 1 || interval > 60 {
			return errors.New("poll_interval must be between 1 and 60")
		}
		cfg.Cursor.PollInterval = interval
		return nil
	case "claude_code.projects_dir", "claude-code.projects-dir", "projects.dir", "projects_dir", "projects-dir":
		path := ExpandHome(strings.TrimSpace(rawValue))
		cfg.ClaudeCode.ProjectsDir = path
		return nil
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
}

func normalizeKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.ReplaceAll(key, "-", "_")
	return key
}

func parseTags(input string) []string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return []string{}
	}
	if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
		var tags []string
		if err := json.Unmarshal([]byte(trimmed), &tags); err == nil {
			return tags
		}
	}
	return splitComma(trimmed)
}

func ExpandHome(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}

func Write(path string, cfg Config) error {
	if err := ensureTabsDir(); err != nil {
		return err
	}

	content := Format(cfg)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		return err
	}
	return nil
}

func ensureTabsDir() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	tabsDir := filepath.Join(home, ".tabs")
	if err := os.MkdirAll(tabsDir, 0o700); err != nil {
		return err
	}
	return nil
}

func Format(cfg Config) string {
	var b strings.Builder
	b.WriteString("# tabs configuration file\n")
	b.WriteString("# Generated by: tabs-cli config\n\n")

	b.WriteString("[local]\n")
	fmt.Fprintf(&b, "ui_port = %d\n", cfg.Local.UIPort)
	fmt.Fprintf(&b, "log_level = %q\n", cfg.Local.LogLevel)
	fmt.Fprintf(&b, "empty_session_retention_hours = %d\n\n", cfg.Local.EmptySessionRetentionHours)

	b.WriteString("[remote]\n")
	fmt.Fprintf(&b, "server_url = %q\n", cfg.Remote.ServerURL)
	fmt.Fprintf(&b, "api_key = %q\n", cfg.Remote.APIKey)
	fmt.Fprintf(&b, "auto_push = %t\n", cfg.Remote.AutoPush)
	fmt.Fprintf(&b, "default_tags = %s\n\n", formatStringArray(cfg.Remote.DefaultTags))

	b.WriteString("[cursor]\n")
	fmt.Fprintf(&b, "db_path = %q\n", cfg.Cursor.DBPath)
	fmt.Fprintf(&b, "poll_interval = %d\n\n", cfg.Cursor.PollInterval)

	b.WriteString("[claude_code]\n")
	fmt.Fprintf(&b, "projects_dir = %q\n", cfg.ClaudeCode.ProjectsDir)

	return b.String()
}

func formatStringArray(values []string) string {
	if len(values) == 0 {
		return "[]"
	}
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, strconv.Quote(value))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
