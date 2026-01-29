package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/victorarias/tabs/internal/logging"
	"github.com/victorarias/tabs/internal/server"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

func main() {
	logger := logging.New(os.Getenv("LOG_LEVEL"), os.Stdout).With("component", "server")

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			fmt.Printf("tabs-server %s (commit: %s, built: %s)\n", Version, Commit, BuildTime)
			return
		case "migrate":
			if err := runMigrations(); err != nil {
				logger.Error("migrate failed", "error", err)
				os.Exit(1)
			}
			logger.Info("migrations applied")
			return
		}
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		logger.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := server.OpenDB(ctx, dbURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if envBool(os.Getenv("MIGRATE_ON_START")) {
		if err := server.RunMigrations(ctx, db, "migrations"); err != nil {
			logger.Error("migration failed", "error", err)
			os.Exit(1)
		}
	}

	port := parsePort(os.Getenv("PORT"), 8080)
	baseURL := strings.TrimSpace(os.Getenv("BASE_URL"))
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://localhost:%d", port)
	}

	authCfg := server.AuthConfig{
		Mode:          os.Getenv("AUTH_MODE"),
		HeaderUser:    os.Getenv("AUTH_HEADER_USER"),
		IAPAudience:   os.Getenv("IAP_AUDIENCE"),
		IAPIssuer:     os.Getenv("IAP_ISSUER"),
		IAPJWKSURL:    os.Getenv("IAP_JWKS_URL"),
		IAPHTTPClient: nil,
	}
	auth, err := server.NewAuthenticator(authCfg)
	if err != nil {
		logger.Error("auth config invalid", "error", err)
		os.Exit(1)
	}

	srv := server.NewServer(db, baseURL, logger, auth)
	addr := ":" + strconv.Itoa(port)
	logger.Info("starting", "version", Version, "commit", Commit, "built", BuildTime)
	logger.Info("listening", "addr", addr)
	if err := srv.ListenAndServe(ctx, addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
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
