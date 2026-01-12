// Package monitor provides Kubernetes deployment rollout monitoring functionality.
package monitor

import (
	"errors"
	"regexp"

	"github.com/ivoronin/kubectl-watch-rollout/internal/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// ErrProgressDeadlineExceeded indicates rollout failed due to progress deadline
var ErrProgressDeadlineExceeded = errors.New("progress deadline exceeded")

const (
	// DefaultPollIntervalSeconds is chosen to balance responsiveness with API load
	DefaultPollIntervalSeconds = 5
	// DefaultMaxEvents limits output while showing most common issues
	DefaultMaxEvents = 10
	// DefaultSimilarityThreshold for Drain algorithm.
	// Minimum ratio of matching tokens for messages to cluster.
	// Lower = more aggressive clustering. Default 0.5 = 50% tokens must match.
	DefaultSimilarityThreshold = 0.5
	// DefaultProgressBarWidth fits standard 80-column terminals with margin
	DefaultProgressBarWidth = 70
	// DefaultProgressDeadlineSeconds is the K8s default progress deadline
	DefaultProgressDeadlineSeconds = 600
	// DefaultMaxSurge is the K8s default for RollingUpdate strategy
	DefaultMaxSurge = "25%"
	// DefaultMaxUnavailable is the K8s default for RollingUpdate strategy
	DefaultMaxUnavailable = "25%"
	// MinProgressForETA is the minimum progress ratio for meaningful ETA calculations
	MinProgressForETA = 0.05
	// MaxRealisticETAHours caps ETA predictions to prevent absurd estimates
	MaxRealisticETAHours = 24
)

// Config holds configuration parameters for the rollout monitor.
// Use DefaultConfig() to obtain sensible defaults, then override as needed.
type Config struct {
	PollIntervalSeconds int
	MaxEvents           int
	ProgressBarWidth    int
	SimilarityThreshold float64        // Controls event clustering (0.0-1.0, lower = more aggressive)
	UntilComplete       bool           // Exit after monitoring one rollout (default: continuous)
	LineMode            bool           // Use line-based output for CI/CD (default: TUI mode)
	IgnoreEvents        *regexp.Regexp // Regex to filter out events by "Reason: Message"
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return Config{
		PollIntervalSeconds: DefaultPollIntervalSeconds,
		MaxEvents:           DefaultMaxEvents,
		ProgressBarWidth:    DefaultProgressBarWidth,
		SimilarityThreshold: DefaultSimilarityThreshold,
		UntilComplete:       false, // Default: continuous monitoring
		IgnoreEvents:        nil,
	}
}

// RolloutResult represents the outcome of a monitoring iteration.
// Done is true when rollout finished. Failed is true only when deadline exceeded.
type RolloutResult struct {
	Done   bool
	Failed bool
}

// isDeploymentComplete checks if deployment rollout is complete
func isDeploymentComplete(status appsv1.DeploymentStatus) bool {
	return hasCondition(status, appsv1.DeploymentAvailable, corev1.ConditionTrue, "") &&
		hasCondition(status, appsv1.DeploymentProgressing, corev1.ConditionTrue, NewReplicaSetAvailable)
}

// isDeploymentFailed checks if deployment rollout has failed
func isDeploymentFailed(status appsv1.DeploymentStatus) bool {
	return hasCondition(status, appsv1.DeploymentProgressing, corev1.ConditionFalse, ProgressDeadlineExceeded)
}

// CalculateRolloutStatus determines rollout status from deployment conditions.
// Returns Complete if fully available, DeadlineExceeded if failed, otherwise Progressing.
func CalculateRolloutStatus(deployment *appsv1.Deployment) types.RolloutStatus {
	status := deployment.Status

	if isDeploymentComplete(status) {
		return types.StatusComplete
	}

	if isDeploymentFailed(status) {
		return types.StatusDeadlineExceeded
	}

	return types.StatusProgressing
}

// hasCondition checks if a specific condition exists with given type and status.
// If reason is empty, only type/status checked. If provided, all three must match.
func hasCondition(
	status appsv1.DeploymentStatus,
	condType appsv1.DeploymentConditionType,
	condStatus corev1.ConditionStatus,
	reason string,
) bool {
	for _, c := range status.Conditions {
		if c.Type == condType && c.Status == condStatus {
			if reason == "" || c.Reason == reason {
				return true
			}
		}
	}

	return false
}

// getInt32OrDefault returns the value or a default if nil
func getInt32OrDefault(val *int32, defaultVal int32) int32 {
	if val == nil {
		return defaultVal
	}

	return *val
}
