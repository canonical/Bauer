package main

import (
	"bauer/cmd/app/core/middleware"
	"bauer/cmd/app/types"
	v1 "bauer/cmd/app/v1"
	"bauer/internal/orchestrator"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
)

func run() error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
	slog.Info("startup", "status", "initializing API")
	defer slog.Info("shutdown complete")

	orchestrator := orchestrator.NewOrchestrator()
	cfg, err := types.LoadConfig()
	if err != nil {
		slog.Error("failed to load config", "error", err.Error())
		return err
	}

	if cfg.TargetRepo != "" {
		// Convert output directory to absolute before changing directory
		absOutputDir, err := filepath.Abs(cfg.BaseOutputDir)
		if err != nil {
			return fmt.Errorf("failed to resolve output directory path: %w", err)
		}
		cfg.BaseOutputDir = absOutputDir

		if err := os.Chdir(cfg.TargetRepo); err != nil {
			return fmt.Errorf("failed to change to target repository %q: %w", cfg.TargetRepo, err)
		}
		cwd, _ := os.Getwd()
		slog.Info("Working directory", "path", cwd)
	}

	rc := types.RouteConfig{
		APIConfig:    *cfg,
		Orchestrator: orchestrator,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/job", v1.JobPost(rc))
	mux.HandleFunc("/api/v1/health", v1.GetHealth)
	handler := middleware.RequestTrace(middleware.APIBasicAuth(mux, cfg.APISecret))
	slog.Info("starting server", "address", ":8090")
	err = http.ListenAndServe(":8090", handler)

	if err != nil {
		slog.Error("server error", "error", err.Error())
		slog.Info("shutdown complete with errors")
		return err
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}
