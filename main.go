// Bingo is a self-hosted bingo card server: JSON API, embedded Vue front end, SQLite storage.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bingo/internal/api"
	"bingo/internal/store"
	"bingo/web"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	addr := flag.String("addr", envOr("BINGO_ADDR", ":8080"), "listen address (env BINGO_ADDR)")
	dbPath := flag.String("db", envOr("BINGO_DB", "bingo.db"), "path to the SQLite database (env BINGO_DB)")
	flag.Parse()

	if err := run(*addr, *dbPath); err != nil {
		slog.Error("fatal error", "err", err)
		os.Exit(1)
	}
}

func run(addr, dbPath string) error {
	st, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer st.Close()

	srv := &http.Server{
		Handler:           api.New(st, web.Dist()),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	slog.Info("bingo started", "addr", ln.Addr().String(), "db", dbPath)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
