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
	case "push", "push-session":
		err = runPush(args)
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
	fmt.Println("  tabs-cli push --session-id <id> --tool <tool> [--tag key:value]")
	fmt.Println("  tabs-cli status")
	fmt.Println("  tabs-cli ui")
	fmt.Println("  tabs-cli config --set key=value")
	fmt.Println("\nCommands:")
	fmt.Println("  capture        Send hook event to daemon")
	fmt.Println("  install        Install Claude Code hook scripts")
	fmt.Println("  push           Upload a session to remote server")
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

type pushTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
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
		} else if tool == "cursor" {
			if existing, ok := eventObj["conversation_id"].(string); ok && existing != "" {
				sessionID = existing
			}
		}
		if sessionID == "" {
			return errors.New("--session-id is required (or event.session_id / event.conversation_id)")
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
	if tool == "cursor" {
		if existing, ok := eventObj["conversation_id"].(string); ok && existing != "" && existing != sessionID {
			return fmt.Errorf("event.conversation_id (%s) does not match --session-id", existing)
		}
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

type tagFlags []string

func (t *tagFlags) String() string {
	return strings.Join(*t, ", ")
}

func (t *tagFlags) Set(value string) error {
	*t = append(*t, value)
	return nil
}

func runPush(args []string) error {
	fs := flag.NewFlagSet("push", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var sessionID string
	var tool string
	var tags tagFlags

	fs.StringVar(&sessionID, "session-id", "", "Session ID (UUID)")
	fs.StringVar(&tool, "tool", "claude-code", "Tool name: claude-code or cursor")
	fs.Var(&tags, "tag", "Tag key:value (repeatable)")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if sessionID == "" {
		return errors.New("--session-id is required")
	}
	if tool != "claude-code" && tool != "cursor" {
		return errors.New("--tool must be claude-code or cursor")
	}

	parsedTags, err := parsePushTags(tags)
	if err != nil {
		return err
	}

	resp, err := sendSocketRequest(request{
		Version: protocolVersion,
		Type:    "push_session",
		Payload: map[string]interface{}{
			"session_id": sessionID,
			"tool":       tool,
			"tags":       parsedTags,
		},
	})
	if err != nil {
		return err
	}

	if resp.Status != "ok" {
		return formatResponseError(resp)
	}

	var data struct {
		RemoteID string `json:"remote_id"`
		URL      string `json:"url"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		fmt.Println(string(resp.Data))
		return nil
	}

	if data.RemoteID != "" {
		fmt.Printf("Session uploaded: %s\n", data.RemoteID)
	}
	if data.URL != "" {
		fmt.Printf("URL: %s\n", data.URL)
	}
	if data.RemoteID == "" && data.URL == "" {
		fmt.Println("Session uploaded.")
	}
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

	claudeCommand := "tabs-cli capture-event --tool=claude-code"
	cursorCommand := "tabs-cli capture-event --tool=cursor"

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

	claudeSettingsPath, err := installClaudeSettings(claudeCommand)
	if err != nil {
		return err
	}
	cursorConfigPath, err := installCursorHooks(cursorCommand)
	if err != nil {
		return err
	}

	fmt.Printf("Installed Claude Code hooks in %s\n", hooksDir)
	for _, path := range installed {
		fmt.Printf(" - %s\n", path)
	}
	if claudeSettingsPath != "" {
		fmt.Printf("Updated Claude settings: %s\n", claudeSettingsPath)
	}
	if cursorConfigPath != "" {
		fmt.Printf("Updated Cursor hooks: %s\n", cursorConfigPath)
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

type claudeHookGroup struct {
	Matcher string            `json:"matcher,omitempty"`
	Hooks   []claudeHookEntry `json:"hooks"`
}

type claudeHookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

func installClaudeSettings(command string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o700); err != nil {
		return "", err
	}

	settings := make(map[string]interface{})
	if data, err := os.ReadFile(settingsPath); err == nil && len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &settings); err != nil {
			return "", fmt.Errorf("parse %s: %w", settingsPath, err)
		}
	}

	hooks, _ := settings["hooks"].(map[string]interface{})
	if hooks == nil {
		hooks = make(map[string]interface{})
	}

	events := []string{"SessionStart", "UserPromptSubmit", "Stop"}
	for _, event := range events {
		hooks[event] = ensureClaudeSettingsHook(hooks[event], command)
	}
	settings["hooks"] = hooks

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(settingsPath, out, 0o600); err != nil {
		return "", err
	}
	return settingsPath, nil
}

func ensureClaudeSettingsHook(existing interface{}, command string) []claudeHookGroup {
	var groups []claudeHookGroup
	commandExists := false

	// Parse existing hook groups
	if arr, ok := existing.([]interface{}); ok {
		for _, item := range arr {
			if m, ok := item.(map[string]interface{}); ok {
				if hasCommand(m, command) {
					commandExists = true
				}
				group := claudeHookGroup{}
				if matcher, ok := m["matcher"].(string); ok {
					group.Matcher = matcher
				}
				if hooksArr, ok := m["hooks"].([]interface{}); ok {
					for _, h := range hooksArr {
						if hm, ok := h.(map[string]interface{}); ok {
							entry := claudeHookEntry{}
							if t, ok := hm["type"].(string); ok {
								entry.Type = t
							}
							if c, ok := hm["command"].(string); ok {
								entry.Command = c
							}
							group.Hooks = append(group.Hooks, entry)
						}
					}
				}
				groups = append(groups, group)
			}
		}
	}

	// Add new hook group if command doesn't exist
	if !commandExists {
		groups = append(groups, claudeHookGroup{
			Hooks: []claudeHookEntry{{Type: "command", Command: command}},
		})
	}

	return groups
}

func hasCommand(group map[string]interface{}, command string) bool {
	hooksArr, ok := group["hooks"].([]interface{})
	if !ok {
		return false
	}
	for _, h := range hooksArr {
		if hm, ok := h.(map[string]interface{}); ok {
			if c, ok := hm["command"].(string); ok && strings.TrimSpace(c) == command {
				return true
			}
		}
	}
	return false
}

type cursorHooks struct {
	Version int                     `json:"version"`
	Hooks   map[string][]cursorHook `json:"hooks"`
}

type cursorHook struct {
	Command string `json:"command"`
}

func installCursorHooks(command string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configPath := filepath.Join(home, ".cursor", "hooks.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		return "", err
	}

	cfg := cursorHooks{Version: 1, Hooks: map[string][]cursorHook{}}
	if data, err := os.ReadFile(configPath); err == nil && len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return "", fmt.Errorf("parse %s: %w", configPath, err)
		}
	}
	if cfg.Hooks == nil {
		cfg.Hooks = map[string][]cursorHook{}
	}
	cfg.Version = 1
	events := []string{"beforeSubmitPrompt", "stop"}
	for _, event := range events {
		cfg.Hooks[event] = ensureCursorHook(cfg.Hooks[event], command)
	}

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(configPath, out, 0o600); err != nil {
		return "", err
	}
	return configPath, nil
}

func ensureCursorHook(existing []cursorHook, command string) []cursorHook {
	for _, hook := range existing {
		if strings.TrimSpace(hook.Command) == command {
			return existing
		}
	}
	return append(existing, cursorHook{Command: command})
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

	cfgPath, err := cfgpkg.Path()
	if err != nil {
		return err
	}

	cfg, err := cfgpkg.Load(cfgPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		cfg = cfgpkg.Default()
	}

	for _, set := range sets {
		key, value, err := splitKeyValue(set)
		if err != nil {
			return err
		}
		if err := cfgpkg.ApplySet(&cfg, key, value); err != nil {
			return err
		}
	}

	if err := cfgpkg.Write(cfgPath, cfg); err != nil {
		return err
	}

	fmt.Printf("Updated %s\n", cfgPath)
	return nil
}

func showConfig() error {
	cfgPath, err := cfgpkg.Path()
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

func parsePushTags(values []string) ([]pushTag, error) {
	var tags []pushTag
	for _, raw := range values {
		for _, entry := range splitComma(raw) {
			tag, err := parseTagEntry(entry)
			if err != nil {
				return nil, err
			}
			if tag.Key == "" || tag.Value == "" {
				continue
			}
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

func parseTagEntry(raw string) (pushTag, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return pushTag{}, nil
	}
	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) != 2 {
		parts = strings.SplitN(trimmed, "=", 2)
	}
	if len(parts) != 2 {
		return pushTag{}, fmt.Errorf("invalid tag %q, expected key:value", trimmed)
	}
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	if key == "" || value == "" {
		return pushTag{}, fmt.Errorf("invalid tag %q, empty key or value", trimmed)
	}
	return pushTag{Key: key, Value: value}, nil
}

func splitComma(input string) []string {
	parts := strings.Split(input, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	return parts
}
