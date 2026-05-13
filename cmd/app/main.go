package main

import (
	"bauer/cmd/app/core/middleware"
	"bauer/cmd/app/types"
	v1 "bauer/cmd/app/v1"
	"bauer/internal/artifacts"
	"bauer/internal/copilotcli"
	"bauer/internal/orchestrator"
	"bauer/internal/source"
	"bauer/internal/workflow"
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

	cwd, err := os.Getwd()
	if err != nil {
		slog.Error("failed to get working directory", "error", err.Error())
		return err
	}

	copilotClient, err := copilotcli.NewClient(cwd)
	if err != nil {
		slog.Error("failed to create Copilot client", "error", err.Error())
		return err
	}

	gdocsAdapter := source.NewGDocsAdapter()
	sources := source.NewManager(gdocsAdapter)

	// API server resolves artifacts dir from env only (no CLI flag for the API)
	resolvedArtifactsDir := os.Getenv("BAUER_ARTIFACTS_DIR")
	if resolvedArtifactsDir == "" {
		resolvedArtifactsDir = "bauer-artifacts"
	}
	artMgr := artifacts.NewManager(resolvedArtifactsDir)

	orchestrator := orchestrator.NewOrchestrator(copilotClient, sources, artMgr)
	cfg, err := types.LoadConfig()
	if err != nil {
		slog.Error("failed to load config", "error", err.Error())
		return err
	}

	rc := types.RouteConfig{
		APIConfig:    *cfg,
		Orchestrator: orchestrator,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/job", v1.JobPost(rc))
	mux.HandleFunc("/api/v1/health", v1.GetHealth)
	mux.HandleFunc("/api/v1/workflow", workflow.ExecuteWorkflowHandler(orchestrator))
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
