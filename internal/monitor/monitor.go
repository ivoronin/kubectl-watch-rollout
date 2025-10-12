// Package monitor provides Kubernetes deployment rollout monitoring functionality.
//
// Architecture (MVC Pattern):
//   - Controller: Orchestrates monitoring logic and state management
//   - View: Presentation layer interface (see view.go)
//   - ConsoleView: Terminal implementation with Renderer + TerminalController
//   - Renderer: Output formatting and display (see renderer.go)
//   - DeploymentRepository: Data access layer for Kubernetes API (see repository.go)
//   - Types: Domain models and DTOs (see types.go)
//   - Metrics: Business logic for rollout calculations (see metrics.go)
//
// Data Flow:
//
//	Repository (K8s API) → Controller → Model (RolloutSnapshot) → View (ConsoleView)
package monitor

import (
	"context"
	"fmt"
	"os"
	"time"
)

// Controller handles deployment rollout monitoring (Controller layer).
// It orchestrates between repository (data), view (presentation), and metrics (logic).
type Controller struct {
	repo           *DeploymentRepository
	view           View
	deploymentName string
	config         Config
}

// New creates a new Controller instance for monitoring a deployment rollout
func New(repo *DeploymentRepository, deploymentName string) (*Controller, error) {
	return NewWithConfig(repo, deploymentName, DefaultConfig())
}

// NewWithConfig creates a new Controller instance with custom configuration
func NewWithConfig(repo *DeploymentRepository, deploymentName string, config Config) (*Controller, error) {
	if repo == nil {
		return nil, fmt.Errorf("internal error: repository is required")
	}
	if deploymentName == "" {
		return nil, fmt.Errorf("deployment name is required")
	}

	return &Controller{
		repo:           repo,
		view:           NewConsoleView(config, os.Stdout),
		deploymentName: deploymentName,
		config:         config,
	}, nil
}

// Run starts monitoring the deployment and returns error if monitoring fails
func (c *Controller) Run(ctx context.Context) error {
	defer c.view.Shutdown()

	pollInterval := time.Duration(c.config.PollIntervalSeconds) * time.Second
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		result, err := c.processDeployment(ctx)
		if err != nil {
			return err
		}

		if result.Done {
			// Only exit if --until-complete flag is set
			if c.config.UntilComplete {
				if result.Failed {
					return ErrProgressDeadlineExceeded
				}
				return nil
			}
			// Default: continuous monitoring - continue loop
		}

		select {
		case <-ctx.Done():
			fmt.Println("\n\nMonitoring interrupted by user")
			return fmt.Errorf("monitoring cancelled")
		case <-ticker.C:
		}
	}
}

// processDeployment fetches deployment data, builds snapshot, and renders view.
// Returns rollout result indicating done/failed state, or error if processing fails.
func (c *Controller) processDeployment(ctx context.Context) (RolloutResult, error) {
	snapshot, err := c.buildSnapshot(ctx)
	if err != nil {
		return RolloutResult{}, fmt.Errorf("failed to build snapshot: %w", err)
	}

	c.view.RenderSnapshot(snapshot)

	return RolloutResult{
		Done:   snapshot.Status.IsDone(),
		Failed: snapshot.Status.IsFailed(),
	}, nil
}

