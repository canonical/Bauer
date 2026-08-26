package main

import (
	"bauer/cmd/bauer-api/core/middleware"
	"bauer/cmd/bauer-api/types"
	v1 "bauer/cmd/bauer-api/v1"
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

	// The go-framework charm injects the port the workload should listen on via
	// the APP_PORT environment variable. Fall back to the go-framework default
	// of 8080 when it is not set (e.g. running outside the charm).
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	// Starting web server.
	address := ":" + port
	slog.Info("starting server", "address", address)
	err = http.ListenAndServe(address, middleware.RequestTrace(mux))

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
