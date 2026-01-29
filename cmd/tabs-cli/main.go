package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	cfgpkg "github.com/victorarias/tabs/internal/config"
	"github.com/victorarias/tabs/internal/daemon"
	"github.com/victorarias/tabs/internal/localserver"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

const protocolVersion = "1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "capture", "capture-event":
		err = runCapture(args)
	case "install":
		err = runInstall(args)
	case "status":
		err = runStatus(args)
	case "ui":
		err = runUI(args)
	case "config":
		err = runConfig(args)
	case "version", "--version", "-version", "-v":
		printVersion()
		return
	case "help", "-h", "--help":
		printUsage()
		return
	default:
		printUsage()
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf("tabs-cli %s (commit: %s, built: %s)\n", Version, Commit, BuildTime)
	fmt.Println("\nUsage:")
	fmt.Println("  tabs-cli capture --session-id <id> --event <json> [--tool claude-code]")
	fmt.Println("  tabs-cli install")
	fmt.Println("  tabs-cli status")
	fmt.Println("  tabs-cli ui")
	fmt.Println("  tabs-cli config --set key=value")
	fmt.Println("\nCommands:")
	fmt.Println("  capture        Send hook event to daemon")
	fmt.Println("  install        Install Claude Code hook scripts")
	fmt.Println("  status         Show daemon status")
	fmt.Println("  ui             Run local web UI API server")
	fmt.Println("  config         Manage configuration")
	fmt.Println("  version        Print version info")
}

func printVersion() {
	fmt.Printf("tabs-cli %s (commit: %s, built: %s)\n", Version, Commit, BuildTime)
}

type request struct {
	Version string      `json:"version"`
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type response struct {
	Version string          `json:"version"`
	Status  string          `json:"status"`
	Data    json.RawMessage `json:"data"`
	Error   *responseError  `json:"error"`
}

type responseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func runCapture(args []string) error {
	fs := flag.NewFlagSet("capture", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var sessionID string
	var eventRaw string
	var tool string
	var timestamp string

	fs.StringVar(&sessionID, "session-id", "", "Session ID (UUID)")
	fs.StringVar(&eventRaw, "event", "", "Event JSON string, '-' for stdin, or '@file' to read")
	fs.StringVar(&tool, "tool", "claude-code", "Tool name: claude-code or cursor")
	fs.StringVar(&timestamp, "timestamp", "", "ISO 8601 timestamp (default: now)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	eventObj, err := readEventObject(eventRaw)
	if err != nil {
		return err
	}

	if sessionID == "" {
		if existing, ok := eventObj["session_id"].(string); ok && existing != "" {
			sessionID = existing
		} else {
			return errors.New("--session-id is required (or event.session_id)")
		}
	}

	if existing, ok := eventObj["session_id"]; ok {
		existingStr, ok := existing.(string)
		if !ok {
			return errors.New("event.session_id must be a string")
		}
		if existingStr != sessionID {
			return fmt.Errorf("event.session_id (%s) does not match --session-id", existingStr)
		}
	} else {
		eventObj["session_id"] = sessionID
	}

	ts := timestamp
	if ts == "" {
		ts = time.Now().UTC().Format(time.RFC3339Nano)
	}

	payload := map[string]interface{}{
		"tool":      tool,
		"timestamp": ts,
		"event":     eventObj,
	}

	resp, err := sendSocketRequest(request{
		Version: protocolVersion,
		Type:    "capture_event",
		Payload: payload,
	})
	if err != nil {
		return err
	}

	if resp.Status != "ok" {
		return formatResponseError(resp)
	}

	var data struct {
		SessionID     string `json:"session_id"`
		EventsWritten int    `json:"events_written"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		fmt.Println(string(resp.Data))
		return nil
	}

	fmt.Printf("Captured session %s (%d events)\n", data.SessionID, data.EventsWritten)
	return nil
}

func runInstall(args []string) error {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var force bool
	fs.BoolVar(&force, "force", false, "Overwrite existing hook scripts")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("install does not take arguments")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	hooksDir := filepath.Join(home, ".claude", "hooks")
	if err := os.MkdirAll(hooksDir, 0o700); err != nil {
		return err
	}

	scripts := map[string]string{
		"on-project-start.sh":      claudeHookScript,
		"on-user-prompt-submit.sh": claudeHookScript,
	}

	installed := make([]string, 0, len(scripts))
	for name, content := range scripts {
		path := filepath.Join(hooksDir, name)
		if !force {
			if existing, err := os.ReadFile(path); err == nil {
				if string(existing) == content {
					installed = append(installed, path)
					continue
				}
				return fmt.Errorf("%s already exists (use --force to overwrite)", path)
			}
		}
		if err := writeExecutable(path, content, 0o755); err != nil {
			return err
		}
		installed = append(installed, path)
	}

	fmt.Printf("Installed Claude Code hooks in %s\n", hooksDir)
	for _, path := range installed {
		fmt.Printf(" - %s\n", path)
	}
	return nil
}

const claudeHookScript = "#!/usr/bin/env bash\nset -euo pipefail\n\nif ! command -v tabs-cli >/dev/null 2>&1; then\n  exit 0\nfi\n\nexec tabs-cli capture-event --tool=claude-code\n"

func writeExecutable(path, content string, perm os.FileMode) error {
	if err := os.WriteFile(path, []byte(content), perm); err != nil {
		return err
	}
	return os.Chmod(path, perm)
}

func runStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("status does not take arguments")
	}

	resp, err := sendSocketRequest(request{
		Version: protocolVersion,
		Type:    "daemon_status",
		Payload: map[string]interface{}{},
	})
	if err != nil {
		return err
	}

	if resp.Status != "ok" {
		return formatResponseError(resp)
	}

	var data struct {
		PID              int    `json:"pid"`
		UptimeSeconds    int    `json:"uptime_seconds"`
		SessionsCaptured int    `json:"sessions_captured"`
		EventsProcessed  int    `json:"events_processed"`
		CursorPolling    bool   `json:"cursor_polling"`
		LastEventAt      string `json:"last_event_at"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		fmt.Println(string(resp.Data))
		return nil
	}

	fmt.Printf("Daemon running (pid %d)\n", data.PID)
	fmt.Printf("Uptime: %ds\n", data.UptimeSeconds)
	fmt.Printf("Sessions captured: %d\n", data.SessionsCaptured)
	fmt.Printf("Events processed: %d\n", data.EventsProcessed)
	fmt.Printf("Cursor polling: %t\n", data.CursorPolling)
	if data.LastEventAt != "" {
		fmt.Printf("Last event: %s\n", data.LastEventAt)
	}
	return nil
}

func runUI(args []string) error {
	fs := flag.NewFlagSet("ui", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("ui does not take arguments")
	}

	cfgPath, err := cfgpkg.Path()
	if err != nil {
		return err
	}
	cfg, err := cfgpkg.Load(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			cfg = cfgpkg.Default()
		} else {
			return err
		}
	}

	baseDir, err := daemon.EnsureBaseDir()
	if err != nil {
		return err
	}

	server := localserver.NewServer(baseDir, cfg)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	fmt.Printf("Local API running at http://127.0.0.1:%d\n", cfg.Local.UIPort)
	return server.ListenAndServe(ctx)
}

type setFlags []string

func (s *setFlags) String() string {
	return strings.Join(*s, ", ")
}

func (s *setFlags) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func runConfig(args []string) error {
	fs := flag.NewFlagSet("config", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var sets setFlags
	fs.Var(&sets, "set", "Set configuration key=value")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if len(sets) == 0 && fs.NArg() > 0 {
		sub := fs.Arg(0)
		switch sub {
		case "set":
			if fs.NArg() < 3 {
				return errors.New("config set requires key and value")
			}
			key := fs.Arg(1)
			value := fs.Arg(2)
			sets = append(sets, fmt.Sprintf("%s=%s", key, value))
		case "show":
			return showConfig()
		default:
			return fmt.Errorf("unknown config command: %s", sub)
		}
	}

	if len(sets) == 0 {
		return errors.New("config requires --set key=value or 'config set <key> <value>'")
	}

	cfgPath, err := configFilePath()
	if err != nil {
		return err
	}

	cfg, err := loadConfig(cfgPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		cfg = defaultConfig()
	}

	for _, set := range sets {
		key, value, err := splitKeyValue(set)
		if err != nil {
			return err
		}
		if err := applyConfigSet(&cfg, key, value); err != nil {
			return err
		}
	}

	if err := writeConfig(cfgPath, cfg); err != nil {
		return err
	}

	fmt.Printf("Updated %s\n", cfgPath)
	return nil
}

func showConfig() error {
	cfgPath, err := configFilePath()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return err
	}
	fmt.Print(string(data))
	return nil
}

func configFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".tabs", "config.toml"), nil
}

func daemonSocketPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".tabs", "daemon.sock"), nil
}

func sendSocketRequest(req request) (*response, error) {
	path, err := daemonSocketPath()
	if err != nil {
		return nil, err
	}

	conn, err := dialDaemon(path)
	if err != nil {
		if startErr := ensureDaemonRunning(); startErr != nil {
			return nil, fmt.Errorf("start daemon: %w", startErr)
		}
		conn, err = dialDaemon(path)
		if err != nil {
			return nil, fmt.Errorf("connect daemon: %w", err)
		}
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return nil, err
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	payload = append(payload, '\n')
	if _, err := conn.Write(payload); err != nil {
		return nil, err
	}

	reader := bufio.NewReader(conn)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	var resp response
	if err := json.Unmarshal(bytes.TrimSpace(line), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func dialDaemon(path string) (net.Conn, error) {
	return net.DialTimeout("unix", path, 2*time.Second)
}

func ensureDaemonRunning() error {
	baseDir, err := daemonBaseDir()
	if err != nil {
		return err
	}
	pidPath := filepath.Join(baseDir, "daemon.pid")
	socketPath := filepath.Join(baseDir, "daemon.sock")

	alive, pid, err := daemonPIDAlive(pidPath)
	if err != nil {
		return err
	}
	if alive {
		if err := waitForSocket(socketPath, 2*time.Second); err == nil {
			return nil
		}
		return fmt.Errorf("daemon running (pid %d) but socket unavailable", pid)
	}

	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		return err
	}

	daemonPath, err := findDaemonBinary()
	if err != nil {
		return err
	}

	cmd := exec.Command(daemonPath)
	logFile, err := os.OpenFile(filepath.Join(baseDir, "daemon.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err == nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	} else {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	}

	if err := cmd.Start(); err != nil {
		if logFile != nil {
			_ = logFile.Close()
		}
		return err
	}
	if cmd.Process != nil {
		_ = cmd.Process.Release()
	}
	if logFile != nil {
		_ = logFile.Close()
	}

	return waitForSocket(socketPath, 2*time.Second)
}

func daemonBaseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".tabs"), nil
}

func daemonPIDAlive(pidPath string) (bool, int, error) {
	data, err := os.ReadFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, 0, nil
		}
		return false, 0, err
	}
	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid <= 0 {
		return false, 0, nil
	}
	if err := syscall.Kill(pid, 0); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			return false, 0, nil
		}
		if errors.Is(err, syscall.EPERM) {
			return true, pid, nil
		}
		return false, 0, err
	}
	return true, pid, nil
}

func waitForSocket(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("unix", path, 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("daemon socket did not appear at %s", path)
}

func findDaemonBinary() (string, error) {
	if path, err := exec.LookPath("tabs-daemon"); err == nil {
		return path, nil
	}
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		candidate := filepath.Join(dir, "tabs-daemon")
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", errors.New("tabs-daemon binary not found in PATH")
}

func formatResponseError(resp *response) error {
	if resp.Error == nil {
		return fmt.Errorf("daemon error: %s", resp.Status)
	}
	return fmt.Errorf("daemon error: %s (%s)", resp.Error.Message, resp.Error.Code)
}

func readEventObject(raw string) (map[string]interface{}, error) {
	data, err := readEventBytes(raw)
	if err != nil {
		return nil, err
	}
	var event map[string]interface{}
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("invalid event JSON: %w", err)
	}
	if len(event) == 0 {
		return nil, errors.New("event payload is empty")
	}
	return event, nil
}

func readEventBytes(raw string) ([]byte, error) {
	switch {
	case raw == "" || raw == "-":
		return io.ReadAll(os.Stdin)
	case strings.HasPrefix(raw, "@"):
		path := strings.TrimPrefix(raw, "@")
		return os.ReadFile(path)
	default:
		return []byte(raw), nil
	}
}

func splitKeyValue(input string) (string, string, error) {
	parts := strings.SplitN(input, "=", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid set value %q, expected key=value", input)
	}
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	if key == "" {
		return "", "", fmt.Errorf("invalid set value %q, empty key", input)
	}
	return key, value, nil
}

type Config struct {
	Local      LocalConfig
	Remote     RemoteConfig
	Cursor     CursorConfig
	ClaudeCode ClaudeCodeConfig
}

type LocalConfig struct {
	UIPort   int
	LogLevel string
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

func defaultConfig() Config {
	return Config{
		Local: LocalConfig{
			UIPort:   3787,
			LogLevel: "info",
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

func loadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	return parseConfig(data)
}

func parseConfig(data []byte) (Config, error) {
	cfg := defaultConfig()
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
		if err := applyConfigValue(&cfg, section, key, value); err != nil {
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

func applyConfigValue(cfg *Config, section, key, raw string) error {
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

func applyConfigSet(cfg *Config, key, rawValue string) error {
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
	case "cursor.db_path", "cursor.db-path", "db.path", "db_path", "db-path":
		path := expandHome(strings.TrimSpace(rawValue))
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
		path := expandHome(strings.TrimSpace(rawValue))
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

func expandHome(path string) string {
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

func writeConfig(path string, cfg Config) error {
	if err := ensureTabsDir(); err != nil {
		return err
	}

	content := formatConfig(cfg)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
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
	if err := os.MkdirAll(tabsDir, 0700); err != nil {
		return err
	}
	return nil
}

func formatConfig(cfg Config) string {
	var b strings.Builder
	b.WriteString("# tabs configuration file\n")
	b.WriteString("# Generated by: tabs-cli config\n\n")

	b.WriteString("[local]\n")
	fmt.Fprintf(&b, "ui_port = %d\n", cfg.Local.UIPort)
	fmt.Fprintf(&b, "log_level = %q\n\n", cfg.Local.LogLevel)

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
