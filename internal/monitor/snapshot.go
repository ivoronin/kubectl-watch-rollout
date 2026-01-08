// Package monitor provides Kubernetes deployment rollout monitoring functionality.
//
// This file contains snapshot construction and rollout state calculations.
package monitor

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ReplicaSetState groups pod counts for a ReplicaSet at different lifecycle stages.
// Current is total pods. Ready passed readiness probes. Available for minReadySeconds.
type ReplicaSetState struct {
	Current   int32
	Ready     int32
	Available int32
}

// RolloutSnapshot represents a snapshot of the deployment rollout state
// This is a pure domain DTO with no infrastructure dependencies
type RolloutSnapshot struct {
	// Deployment identification
	DeploymentName string
	NewRSName      string

	// Rollout strategy
	StrategyType   string
	MaxSurge       string
	MaxUnavailable string

	// Pod state tracking (grouped by ReplicaSet)
	Desired int32
	NewRS   ReplicaSetState
	OldRS   ReplicaSetState

	// Progress (0-1 ratios)
	NewProgress float64
	OldProgress float64

	// Time tracking
	StartTime           time.Time
	SnapshotTime        time.Time
	ProgressUpdateTime  *time.Time // lastUpdateTime from Progressing condition
	EstimatedCompletion *time.Time // renamed from ETA for clarity

	// Status and events
	Status RolloutStatus
	Events EventSummary
}

// getProgressUpdateTime extracts LastUpdateTime from Progressing condition.
// Used for deadline calculations as it resets when progress is made. Returns nil if no condition.
func getProgressUpdateTime(deployment *appsv1.Deployment) *time.Time {
	for _, c := range deployment.Status.Conditions {
		if c.Type == appsv1.DeploymentProgressing {
			return &c.LastUpdateTime.Time
		}
	}
	return nil
}

// buildSnapshot constructs a RolloutSnapshot with all calculated data
func (c *Controller) buildSnapshot(ctx context.Context) (*RolloutSnapshot, error) {
	// Fetch Deployment
	deployment, err := c.repo.GetDeployment(ctx, c.deploymentName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch deployment: %w", err)
	}

	// Fetch ReplicaSets
	oldRSs, newRS, err := c.repo.GetReplicaSets(ctx, deployment)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ReplicaSets: %w", err)
	}

	if newRS == nil {
		return nil, fmt.Errorf("no new ReplicaSet found for deployment")
	}

	// Fetch and process events (filtering, clustering, formatting all in one)
	rawEvents, err := c.repo.GetPodEvents(ctx, newRS)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pod events: %w", err)
	}
	events := SummarizeEvents(rawEvents, c.config.IgnoreEvents, c.config.SimilarityThreshold)

	// Extract strategy parameters
	strategy := deployment.Spec.Strategy
	maxSurge := DefaultMaxSurge
	maxUnavailable := DefaultMaxUnavailable
	if strategy.Type == appsv1.RollingUpdateDeploymentStrategyType && strategy.RollingUpdate != nil {
		if strategy.RollingUpdate.MaxSurge != nil {
			maxSurge = formatIntOrPercent(*strategy.RollingUpdate.MaxSurge)
		}
		if strategy.RollingUpdate.MaxUnavailable != nil {
			maxUnavailable = formatIntOrPercent(*strategy.RollingUpdate.MaxUnavailable)
		}
	}

	// Calculate pod states
	desired := getInt32OrDefault(deployment.Spec.Replicas, defaultReplicaCount)

	// newRS is guaranteed to be non-nil at this point
	newRSName := newRS.Name
	newRSState := ReplicaSetState{
		Current:   newRS.Status.Replicas,
		Ready:     newRS.Status.ReadyReplicas,
		Available: newRS.Status.AvailableReplicas,
	}
	startTime := newRS.CreationTimestamp.Time

	oldRSState := ReplicaSetState{}
	for _, rs := range oldRSs {
		oldRSState.Current += rs.Status.Replicas
		oldRSState.Ready += rs.Status.ReadyReplicas
		oldRSState.Available += rs.Status.AvailableReplicas
	}

	// Calculate progress ratios
	var newProgress, oldProgress float64
	var estimatedCompletion *time.Time
	progressUpdateTime := getProgressUpdateTime(deployment)

	if desired > 0 {
		newProgress = float64(newRSState.Available) / float64(desired)
		oldProgress = float64(oldRSState.Available) / float64(desired)

		// Calculate ETA using server timestamps (avoids clock skew issues)
		// startTime and progressUpdateTime are both from K8s server
		if newProgress >= MinProgressForETA && newProgress < 1.0 && progressUpdateTime != nil {
			elapsed := progressUpdateTime.Sub(startTime)
			if elapsed > 0 {
				totalEstimated := time.Duration(float64(elapsed) / newProgress)

				// Only set ETA if realistic (less than MaxRealisticETAHours)
				if totalEstimated < MaxRealisticETAHours*time.Hour {
					eta := startTime.Add(totalEstimated)
					if time.Until(eta) > 0 {
						estimatedCompletion = &eta
					}
				}
			}
		}
	}

	return &RolloutSnapshot{
		DeploymentName:      deployment.Name,
		NewRSName:           newRSName,
		StrategyType:        string(strategy.Type),
		MaxSurge:            maxSurge,
		MaxUnavailable:      maxUnavailable,
		Desired:             desired,
		NewRS:               newRSState,
		OldRS:               oldRSState,
		NewProgress:         newProgress,
		OldProgress:         oldProgress,
		StartTime:           startTime,
		SnapshotTime:        time.Now(),
		ProgressUpdateTime:  getProgressUpdateTime(deployment),
		EstimatedCompletion: estimatedCompletion,
		Status:              CalculateRolloutStatus(deployment, newRS),
		Events:              events,
	}, nil
}

// formatIntOrPercent formats Kubernetes IntOrString for display.
// Returns percentage (e.g., "25%") or decimal (e.g., "3").
func formatIntOrPercent(val intstr.IntOrString) string {
	if val.Type == intstr.String {
		return val.StrVal
	}
	return fmt.Sprintf("%d", val.IntVal)
}
