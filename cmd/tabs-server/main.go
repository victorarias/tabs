package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/victorarias/tabs/internal/server"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			fmt.Printf("tabs-server %s (commit: %s, built: %s)\n", Version, Commit, BuildTime)
			return
		case "migrate":
			if err := runMigrations(); err != nil {
				log.Fatalf("migrate failed: %v", err)
			}
			log.Println("migrations applied")
			return
		}
	}

	logger := log.New(os.Stdout, "", log.LstdFlags|log.LUTC)
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		logger.Fatal("DATABASE_URL is required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := server.OpenDB(ctx, dbURL)
	if err != nil {
		logger.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	if envBool(os.Getenv("MIGRATE_ON_START")) {
		if err := server.RunMigrations(ctx, db, "migrations"); err != nil {
			logger.Fatalf("migration failed: %v", err)
		}
	}

	port := parsePort(os.Getenv("PORT"), 8080)
	baseURL := strings.TrimSpace(os.Getenv("BASE_URL"))
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://localhost:%d", port)
	}

	srv := server.NewServer(db, baseURL, logger)
	addr := ":" + strconv.Itoa(port)
	logger.Printf("tabs-server %s (commit: %s, built: %s)", Version, Commit, BuildTime)
	logger.Printf("listening on %s", addr)
	if err := srv.ListenAndServe(ctx, addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Fatalf("server stopped: %v", err)
	}
}

func runMigrations() error {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := server.OpenDB(ctx, dbURL)
	if err != nil {
		return err
	}
	defer db.Close()
	return server.RunMigrations(ctx, db, "migrations")
}

func parsePort(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	port, err := strconv.Atoi(raw)
	if err != nil || port <= 0 {
		return fallback
	}
	return port
}

func envBool(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
