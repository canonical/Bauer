package types

import (
	"bauer/internal/orchestrator"
)

type RouteConfig struct {
	APIConfig    APIConfig
	Orchestrator orchestrator.Orchestrator
}
