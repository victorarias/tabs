package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/victorarias/tabs/internal/daemon"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

func main() {
	baseDir, err := daemon.EnsureBaseDir()
	if err != nil {
		log.Fatalf("daemon init failed: %v", err)
	}

	logFile, err := os.OpenFile(daemon.LogPath(baseDir), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		log.Fatalf("open log file: %v", err)
	}
	defer logFile.Close()

	logger := log.New(logFile, "", log.LstdFlags|log.LUTC)
	logger.Printf("tabs-daemon %s (commit: %s, built: %s) starting", Version, Commit, BuildTime)

	pidLock, err := daemon.AcquirePID(baseDir)
	if err != nil {
		logger.Fatalf("pid lock failed: %v", err)
	}

	server := daemon.NewServer(baseDir, logger)
	if err := server.Listen(); err != nil {
		_ = pidLock.Release()
		logger.Fatalf("socket listen failed: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(ctx)
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil {
			logger.Printf("server error: %v", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Printf("shutdown error: %v", err)
	}

	if err := pidLock.Release(); err != nil {
		logger.Printf("cleanup error: %v", err)
	}
	logger.Printf("tabs-daemon stopped")
}
