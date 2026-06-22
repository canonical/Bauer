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

	rc := types.RouteConfig{
		APIConfig:    *cfg,
		Orchestrator: orchestrator,
	}

	// Register routes and start the HTTP server.
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("GET /api/v1", v1.GetHealth)

	// Workflow endpoint, which triggers the PR-creation workflow on a target repository.
	mux.HandleFunc("POST /api/v1", v1.WorkflowPost(rc))

	// Starting web server on port 8090
	slog.Info("starting server", "address", ":8090")
	err = http.ListenAndServe(":8090", middleware.RequestTrace(mux))

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
