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

	appsv1 "k8s.io/api/apps/v1"
)

// Controller handles deployment rollout monitoring (Controller layer).
// It orchestrates between repository (data), view (presentation), and metrics (logic).
type Controller struct {
	repo           *DeploymentRepository
	view           View
	deploymentName string
	warnings       map[string]int // Accumulated warnings for current ReplicaSet
	currentNewRS   string         // Name of current new ReplicaSet (for reset detection)
	config         Config
}

// New creates a new Controller instance for monitoring a deployment rollout
func New(repo *DeploymentRepository, deploymentName string) (*Controller, error) {
	if repo == nil {
		return nil, fmt.Errorf("internal error: repository is required")
	}
	if deploymentName == "" {
		return nil, fmt.Errorf("deployment name is required")
	}

	config := DefaultConfig()

	return &Controller{
		repo:           repo,
		view:           NewConsoleView(config, os.Stdout),
		deploymentName: deploymentName,
		warnings:       make(map[string]int),
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
		deployment, err := c.repo.GetDeployment(ctx, c.deploymentName)
		if err != nil {
			return fmt.Errorf("failed to fetch deployment status: %w", err)
		}

		result := c.processDeployment(ctx, deployment)
		if result.Done {
			if result.Failed {
				return ErrProgressDeadlineExceeded
			}
			return nil
		}

		select {
		case <-ctx.Done():
			fmt.Println("\n\nMonitoring interrupted by user")
			return fmt.Errorf("monitoring cancelled")
		case <-ticker.C:
		}
	}
}

// processDeployment evaluates deployment, accumulates warnings, renders view.
// Returns rollout result indicating done/failed state.
func (c *Controller) processDeployment(ctx context.Context, deployment *appsv1.Deployment) RolloutResult {
	oldRSs, newRS, err := c.repo.GetReplicaSets(ctx, deployment)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing ReplicaSets: %v\n", err)
		return RolloutResult{Done: true, Failed: true}
	}

	// Reset state when a new ReplicaSet is detected
	if newRS != nil && newRS.Name != c.currentNewRS {
		c.currentNewRS = newRS.Name
		c.warnings = make(map[string]int)
	}

	c.accumulateWarnings(ctx, newRS)

	snapshot := c.buildSnapshot(deployment, oldRSs, newRS)
	c.view.RenderSnapshot(snapshot)

	return RolloutResult{
		Done:   snapshot.Status.IsDone(),
		Failed: snapshot.Status.IsFailed(),
	}
}

// accumulateWarnings collects and counts warning events from pods
func (c *Controller) accumulateWarnings(ctx context.Context, newRS *appsv1.ReplicaSet) {
	if newRS == nil {
		return
	}

	warnings, err := c.repo.GetPodWarnings(ctx, newRS)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to fetch pod warnings: %v\n", err)
		return
	}

	for _, warning := range warnings {
		c.warnings[warning]++
	}
}
