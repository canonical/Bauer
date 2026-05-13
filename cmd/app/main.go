package main

import (
	"bauer/cmd/app/core/middleware"
	"bauer/cmd/app/types"
	v1 "bauer/cmd/app/v1"
	"bauer/internal/copilotcli"
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

	// Agent factory: creates a fresh Copilot client for each request's target repo.
	// This avoids the bug where a single shared client was bound to the server's
	// startup directory while requests target different cloned repos.
	newAgent := func(cwd string) (orchestrator.Agent, error) {
		return copilotcli.NewClient(cwd)
	}

	cfg, err := types.LoadConfig()
	if err != nil {
		slog.Error("failed to load config", "error", err.Error())
		return err
	}

	rc := types.RouteConfig{
		APIConfig: *cfg,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/job", v1.JobPost(rc, newAgent))
	mux.HandleFunc("/api/v1/health", v1.GetHealth)
	// TODO: refactor workflow handler to use same per-request pattern
	// mux.HandleFunc("/api/v1/workflow", workflow.ExecuteWorkflowHandler(orchestrator))

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
