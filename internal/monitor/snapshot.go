// Package monitor provides Kubernetes deployment rollout monitoring functionality.
//
// This file contains snapshot construction and rollout state calculations.
package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/ivoronin/kubectl-watch-rollout/internal/types"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// getProgressUpdateTime extracts LastUpdateTime from Progressing condition.
func getProgressUpdateTime(deployment *appsv1.Deployment) *time.Time {
	for _, c := range deployment.Status.Conditions {
		if c.Type == appsv1.DeploymentProgressing {
			return &c.LastUpdateTime.Time
		}
	}
	return nil
}

// buildSnapshot constructs a RolloutSnapshot with all calculated data
func (c *Controller) buildSnapshot(ctx context.Context) (*types.RolloutSnapshot, error) {
	deployment, err := c.repo.GetDeployment(ctx, c.deploymentName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch deployment: %w", err)
	}

	oldRSs, newRS, err := c.repo.GetReplicaSets(ctx, deployment)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ReplicaSets: %w", err)
	}

	if newRS == nil {
		return nil, fmt.Errorf("no new ReplicaSet found for deployment")
	}

	rawEvents, err := c.repo.GetPodEvents(ctx, newRS)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pod events: %w", err)
	}
	events := SummarizeEvents(rawEvents, c.config.IgnoreEvents, c.config.SimilarityThreshold)

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

	desired := getInt32OrDefault(deployment.Spec.Replicas, defaultReplicaCount)

	newRSName := newRS.Name
	newRSState := types.ReplicaSetState{
		Current:   newRS.Status.Replicas,
		Ready:     newRS.Status.ReadyReplicas,
		Available: newRS.Status.AvailableReplicas,
	}
	startTime := newRS.CreationTimestamp.Time

	oldRSState := types.ReplicaSetState{}
	for _, rs := range oldRSs {
		oldRSState.Current += rs.Status.Replicas
		oldRSState.Ready += rs.Status.ReadyReplicas
		oldRSState.Available += rs.Status.AvailableReplicas
	}

	var newProgress, oldProgress float64
	var estimatedCompletion *time.Time
	progressUpdateTime := getProgressUpdateTime(deployment)

	if desired > 0 {
		newProgress = float64(newRSState.Available) / float64(desired)
		oldProgress = float64(oldRSState.Available) / float64(desired)

		if newProgress >= MinProgressForETA && newProgress < 1.0 && progressUpdateTime != nil {
			elapsed := progressUpdateTime.Sub(startTime)
			if elapsed > 0 {
				totalEstimated := time.Duration(float64(elapsed) / newProgress)
				if totalEstimated < MaxRealisticETAHours*time.Hour {
					eta := startTime.Add(totalEstimated)
					if time.Until(eta) > 0 {
						estimatedCompletion = &eta
					}
				}
			}
		}
	}

	return &types.RolloutSnapshot{
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
		ProgressUpdateTime:  progressUpdateTime,
		EstimatedCompletion: estimatedCompletion,
		Status:              CalculateRolloutStatus(deployment, newRS),
		Events:              events,
	}, nil
}

func formatIntOrPercent(val intstr.IntOrString) string {
	if val.Type == intstr.String {
		return val.StrVal
	}
	return fmt.Sprintf("%d", val.IntVal)
}
