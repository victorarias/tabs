package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/victorarias/tabs/internal/config"
	"github.com/victorarias/tabs/internal/daemon"
	"github.com/victorarias/tabs/internal/logging"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

func main() {
	fallback := logging.New("info", os.Stderr).With("component", "daemon")
	baseDir, err := daemon.EnsureBaseDir()
	if err != nil {
		fallback.Error("daemon init failed", "error", err)
		os.Exit(1)
	}

	cfg := config.Default()
	if cfgPath, err := config.Path(); err != nil {
		fallback.Warn("config path error", "error", err)
	} else if loaded, err := config.Load(cfgPath); err != nil {
		if !os.IsNotExist(err) {
			fallback.Warn("config load error", "error", err)
		}
	} else {
		cfg = loaded
	}

	logFile, err := os.OpenFile(daemon.LogPath(baseDir), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		fallback.Error("open log file failed", "error", err)
		os.Exit(1)
	}
	defer logFile.Close()

	logger := logging.New(cfg.Local.LogLevel, logFile).With("component", "daemon")
	logger.Info("starting", "version", Version, "commit", Commit, "built", BuildTime)

	pidLock, err := daemon.AcquirePID(baseDir)
	if err != nil {
		logger.Error("pid lock failed", "error", err)
		os.Exit(1)
	}

	server := daemon.NewServer(baseDir, logger)
	if err := server.Listen(); err != nil {
		_ = pidLock.Release()
		logger.Error("socket listen failed", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	daemon.StartCursorPoller(ctx, server, cfg)
	daemon.StartCleanupRoutine(ctx, baseDir, cfg.Local.EmptySessionRetentionHours, logger)

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(ctx)
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil {
			logger.Error("server error", "error", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
	}

	if err := pidLock.Release(); err != nil {
		logger.Error("cleanup error", "error", err)
	}
	logger.Info("stopped")
}
