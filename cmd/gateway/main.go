package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/tejokumar/ai-gateway-mesh/internal/backend"
	"github.com/tejokumar/ai-gateway-mesh/internal/config"
	"github.com/tejokumar/ai-gateway-mesh/internal/gatewayhttp"
	"github.com/tejokumar/ai-gateway-mesh/internal/metrics"
	"github.com/tejokumar/ai-gateway-mesh/internal/router"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/gateway.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		logger.Error("failed to load config", "path", configPath, "error", err)
		os.Exit(1)
	}
	if port := os.Getenv("PORT"); port != "" {
		parsedPort, err := parsePort(port)
		if err != nil {
			logger.Error("invalid PORT", "error", err)
			os.Exit(1)
		}
		cfg.Server.Port = parsedPort
	}

	registry := prometheus.NewRegistry()
	m := metrics.New(registry)
	app := gatewayhttp.New(cfg, router.New(cfg), backend.NewClient(), m, logger)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      app.Handler(),
		ReadTimeout:  durationMS(cfg.Server.ReadTimeoutMS, 30*time.Second),
		WriteTimeout: durationMS(cfg.Server.WriteTimeoutMS, 120*time.Second),
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("starting gateway", "addr", server.Addr, "config_path", configPath)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		logger.Info("shutdown signal received", "signal", sig.String())
	case err := <-errCh:
		if err != nil {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("gateway stopped")
}

func durationMS(value int, fallback time.Duration) time.Duration {
	if value <= 0 {
		return fallback
	}
	return time.Duration(value) * time.Millisecond
}

func parsePort(value string) (int, error) {
	var port int
	if _, err := fmt.Sscanf(value, "%d", &port); err != nil {
		return 0, fmt.Errorf("parse PORT: %w", err)
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("PORT must be between 1 and 65535")
	}
	return port, nil
}
