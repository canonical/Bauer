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

	"github.com/joho/godotenv"
)

func run() error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
	slog.Info("startup", "status", "initializing API")
	defer slog.Info("shutdown complete")

	cfg, err := types.LoadConfig()
	if err != nil {
		slog.Error("failed to load config", "error", err.Error())
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		slog.Error("failed to get working directory", "error", err.Error())
		return err
	}

	copilotAgent, err := copilotcli.NewClient(cwd)
	if err != nil {
		slog.Error("failed to create Copilot client", "error", err.Error())
		return err
	}

	sources := source.NewManager(cfg.CredentialsPath)
	arts := artifacts.NewManager(cfg.ArtifactsDir)
	orch := orchestrator.New(copilotAgent, sources, arts)

	rc := types.RouteConfig{
		APIConfig:    *cfg,
		Orchestrator: orch,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/job", v1.JobPost(rc))
	mux.HandleFunc("/api/v1/health", v1.GetHealth)
	mux.HandleFunc("GET /api/v1/health/ready", v1.ReadinessHandler(cfg))
	mux.HandleFunc("POST /api/v1/workflows", workflow.ExecuteWorkflowHandler(orch))
	mux.HandleFunc("POST /api/v1/issues", v1.IssuesHandler(cfg))
	mux.HandleFunc("POST /api/v1/webhooks/jira", v1.JiraWebhookHandler(cfg))
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
	_ = godotenv.Load(".env")
	_ = godotenv.Load(".env.local")
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}

