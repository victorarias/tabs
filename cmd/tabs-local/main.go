package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/victorarias/tabs/internal/config"
	"github.com/victorarias/tabs/internal/daemon"
	"github.com/victorarias/tabs/internal/localserver"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

func main() {
	baseDir, err := daemon.EnsureBaseDir()
	if err != nil {
		log.Fatalf("local server init failed: %v", err)
	}

	cfgPath, err := config.Path()
	if err != nil {
		log.Fatalf("config path failed: %v", err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			cfg = config.Default()
		} else {
			log.Fatalf("load config failed: %v", err)
		}
	}

	server := localserver.NewServer(baseDir, cfg)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	fmt.Printf("tabs-local %s (commit: %s, built: %s)\n", Version, Commit, BuildTime)
	fmt.Printf("Local API running at http://127.0.0.1:%d\n", cfg.Local.UIPort)

	if err := server.ListenAndServe(ctx); err != nil {
		log.Fatalf("local server stopped: %v", err)
	}
}
