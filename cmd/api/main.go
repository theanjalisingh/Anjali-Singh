package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"futureEnvironsBE/internal/auth"
	"futureEnvironsBE/internal/config"
	"futureEnvironsBE/internal/database"
	"futureEnvironsBE/internal/router"
	"futureEnvironsBE/internal/user"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := newLogger(cfg.LogLevel)
	logger.Info("starting api",
		"service", cfg.AppName,
		"env", cfg.AppEnv,
		"addr", cfg.Addr(),
	)

	dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer dbCancel()

	pool, err := database.NewPool(dbCtx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()
	logger.Info("database connected")

	tokens, err := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTExpiry)
	if err != nil {
		return fmt.Errorf("jwt: %w", err)
	}

	authService := auth.NewService(auth.NewRepository(pool), tokens, logger)
	authHandler := auth.NewHandler(authService)

	userService := user.NewService(user.NewRepository(pool), logger)
	userHandler := user.NewHandler(userService)

	engine := router.New(cfg, logger, tokens, authHandler, userHandler)

	server := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      engine,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// Start the HTTP server in a background goroutine.
	errCh := make(chan error, 1)
	go func() {
		logger.Info("http server listening", "addr", cfg.Addr())
		if listenErr := server.ListenAndServe(); listenErr != nil && !errors.Is(listenErr, http.ErrServerClosed) {
			errCh <- listenErr
		}
		close(errCh)
	}()

	// Wait for OS signal or a listen failure.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-stop:
		logger.Info("shutdown signal received", "signal", sig.String())
	case listenErr := <-errCh:
		if listenErr != nil {
			return fmt.Errorf("http server failed: %w", listenErr)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	logger.Info("shutting down http server", "timeout", cfg.ShutdownTimeout.String())
	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	logger.Info("server stopped cleanly")
	return nil
}

func newLogger(level string) *slog.Logger {
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})
	return slog.New(handler)
}
