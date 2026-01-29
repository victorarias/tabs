package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/victorarias/tabs/internal/config"
	"github.com/victorarias/tabs/internal/daemon"
	"github.com/victorarias/tabs/internal/localserver"
	"github.com/victorarias/tabs/internal/logging"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

func main() {
	fallback := logging.New("info", os.Stderr).With("component", "local")
	baseDir, err := daemon.EnsureBaseDir()
	if err != nil {
		fallback.Error("local server init failed", "error", err)
		os.Exit(1)
	}

	cfgPath, err := config.Path()
	if err != nil {
		fallback.Error("config path failed", "error", err)
		os.Exit(1)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			cfg = config.Default()
		} else {
			fallback.Error("load config failed", "error", err)
			os.Exit(1)
		}
	}
	logger := logging.New(cfg.Local.LogLevel, os.Stdout).With("component", "local")

	server := localserver.NewServer(baseDir, cfg)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	logger.Info("starting", "version", Version, "commit", Commit, "built", BuildTime)
	fmt.Printf("Local API running at http://127.0.0.1:%d\n", cfg.Local.UIPort)

	if err := server.ListenAndServe(ctx); err != nil {
		logger.Error("local server stopped", "error", err)
		os.Exit(1)
	}
}
