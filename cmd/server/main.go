package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/vlasticz/ipcalc/internal/server"
	"github.com/vlasticz/ipcalc/internal/storage"
	"github.com/vlasticz/ipcalc/internal/templates"
)

func main() {
	port := flag.Int("port", envInt("PORT", 8080), "HTTP listen port")
	dbPath := flag.String("db", envStr("DB_PATH", "data/ipcalc.db"), "SQLite database path")
	healthcheck := flag.Bool("healthcheck", false, "Probe /healthz on localhost:<port> and exit 0/1 (for Docker HEALTHCHECK)")
	flag.Parse()

	if *healthcheck {
		os.Exit(probeHealth(*port))
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := run(*port, *dbPath, logger); err != nil {
		logger.Error("server exited with error", "err", err)
		os.Exit(1)
	}
}

func run(port int, dbPath string, logger *slog.Logger) error {
	tmpl, err := templates.Parse()
	if err != nil {
		return fmt.Errorf("parse templates: %w", err)
	}

	store, err := storage.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open storage: %w", err)
	}
	defer store.Close()

	mux := server.New(server.Deps{
		Logger:    logger,
		Templates: tmpl,
		Storage:   store,
		StaticDir: "static",
	})

	addr := net.JoinHostPort("", strconv.Itoa(port))
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", addr, "db", dbPath)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		return err
	}
}

// probeHealth performs an HTTP GET against /healthz on localhost:port and
// returns a process exit code. Used by the Docker HEALTHCHECK directive so
// the image doesn't need curl/wget.
func probeHealth(port int) int {
	client := &http.Client{Timeout: 2 * time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%d/healthz", port)
	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck: %v\n", err)
		return 1
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "healthcheck: status %d\n", resp.StatusCode)
		return 1
	}
	return 0
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
