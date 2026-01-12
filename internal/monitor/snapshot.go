package monitor

// This file contains snapshot construction and rollout state calculations.

import (
	"context"
	"errors"
	"fmt"
	"strconv"
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

// strategyParams holds parsed rolling update strategy parameters.
type strategyParams struct {
	maxSurge       string
	maxUnavailable string
}

// parseStrategyParams extracts maxSurge and maxUnavailable from deployment strategy.
// Returns defaults if strategy is not RollingUpdate or values are not set.
func parseStrategyParams(strategy appsv1.DeploymentStrategy) strategyParams {
	params := strategyParams{
		maxSurge:       DefaultMaxSurge,
		maxUnavailable: DefaultMaxUnavailable,
	}

	if strategy.Type != appsv1.RollingUpdateDeploymentStrategyType || strategy.RollingUpdate == nil {
		return params
	}

	if strategy.RollingUpdate.MaxSurge != nil {
		params.maxSurge = formatIntOrPercent(*strategy.RollingUpdate.MaxSurge)
	}

	if strategy.RollingUpdate.MaxUnavailable != nil {
		params.maxUnavailable = formatIntOrPercent(*strategy.RollingUpdate.MaxUnavailable)
	}

	return params
}

// calculateETA estimates rollout completion time based on current progress.
// Returns nil if ETA cannot be calculated (insufficient progress, complete, or unrealistic estimate).
func calculateETA(progress float64, startTime time.Time, progressUpdateTime *time.Time) *time.Time {
	if progress < MinProgressForETA || progress >= 1.0 {
		return nil
	}

	if progressUpdateTime == nil {
		return nil
	}

	elapsed := progressUpdateTime.Sub(startTime)
	if elapsed <= 0 {
		return nil
	}

	totalEstimated := time.Duration(float64(elapsed) / progress)
	if totalEstimated >= MaxRealisticETAHours*time.Hour {
		return nil
	}

	eta := startTime.Add(totalEstimated)
	if time.Until(eta) <= 0 {
		return nil
	}

	return &eta
}

// aggregateOldRSState sums replica counts across all old ReplicaSets.
func aggregateOldRSState(oldRSs []*appsv1.ReplicaSet) types.ReplicaSetState {
	var state types.ReplicaSetState

	for _, rs := range oldRSs {
		state.Current += rs.Status.Replicas
		state.Ready += rs.Status.ReadyReplicas
		state.Available += rs.Status.AvailableReplicas
	}

	return state
}

// calculateProgress returns the ratio of available to desired replicas, or 0 if desired is 0.
func calculateProgress(available, desired int32) float64 {
	if desired == 0 {
		return 0
	}

	return float64(available) / float64(desired)
}

// buildSnapshot constructs a RolloutSnapshot with all calculated data.
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
		return nil, errors.New("no new ReplicaSet found for deployment")
	}

	rawEvents, err := c.repo.GetPodEvents(ctx, newRS)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pod events: %w", err)
	}

	strategyParams := parseStrategyParams(deployment.Spec.Strategy)
	desired := getInt32OrDefault(deployment.Spec.Replicas, defaultReplicaCount)

	newRSState := types.ReplicaSetState{
		Current:   newRS.Status.Replicas,
		Ready:     newRS.Status.ReadyReplicas,
		Available: newRS.Status.AvailableReplicas,
	}
	oldRSState := aggregateOldRSState(oldRSs)

	newProgress := calculateProgress(newRSState.Available, desired)
	oldProgress := calculateProgress(oldRSState.Available, desired)

	progressUpdateTime := getProgressUpdateTime(deployment)

	return &types.RolloutSnapshot{
		DeploymentName:      deployment.Name,
		NewRSName:           newRS.Name,
		StrategyType:        string(deployment.Spec.Strategy.Type),
		MaxSurge:            strategyParams.maxSurge,
		MaxUnavailable:      strategyParams.maxUnavailable,
		Desired:             desired,
		NewRS:               newRSState,
		OldRS:               oldRSState,
		NewProgress:         newProgress,
		OldProgress:         oldProgress,
		StartTime:           newRS.CreationTimestamp.Time,
		SnapshotTime:        time.Now(),
		ProgressUpdateTime:  progressUpdateTime,
		EstimatedCompletion: calculateETA(newProgress, newRS.CreationTimestamp.Time, progressUpdateTime),
		Status:              CalculateRolloutStatus(deployment),
		Events:              SummarizeEvents(rawEvents, c.config.IgnoreEvents, c.config.SimilarityThreshold),
	}, nil
}

func formatIntOrPercent(val intstr.IntOrString) string {
	if val.Type == intstr.String {
		return val.StrVal
	}

	return strconv.Itoa(int(val.IntVal))
}
