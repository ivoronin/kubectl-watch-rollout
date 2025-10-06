package monitor

import (
	"errors"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// ErrProgressDeadlineExceeded indicates rollout failed due to progress deadline
var ErrProgressDeadlineExceeded = errors.New("progress deadline exceeded")

const (
	// DefaultPollIntervalSeconds is chosen to balance responsiveness with API load
	DefaultPollIntervalSeconds = 5
	// DefaultMaxWarnings limits output while showing most common issues
	DefaultMaxWarnings = 10
	// DefaultProgressBarWidth fits standard 80-column terminals with margin
	DefaultProgressBarWidth = 70
	// DefaultDeadlineWarningThreshold warns when less than 25% time remaining
	DefaultDeadlineWarningThreshold = 0.25
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
	MaxWarnings         int
	ProgressBarWidth    int
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return Config{
		PollIntervalSeconds: DefaultPollIntervalSeconds,
		MaxWarnings:         DefaultMaxWarnings,
		ProgressBarWidth:    DefaultProgressBarWidth,
	}
}

// RolloutStatus represents the current state of a deployment rollout.
// It is an enumeration with values defined as constants below.
type RolloutStatus int

const (
	// StatusProgressing indicates rollout is in progress
	StatusProgressing RolloutStatus = iota
	// StatusDeadlineWarning indicates rollout is approaching deadline
	StatusDeadlineWarning
	// StatusDeadlineExceeded indicates rollout failed due to deadline
	StatusDeadlineExceeded
	// StatusComplete indicates rollout completed successfully
	StatusComplete
)

// IsDone returns true if rollout is complete or failed
func (s RolloutStatus) IsDone() bool {
	return s == StatusComplete || s == StatusDeadlineExceeded
}

// IsFailed returns true if rollout failed
func (s RolloutStatus) IsFailed() bool {
	return s == StatusDeadlineExceeded
}

// WarningEntry represents a warning message observed during rollout.
// Warnings are aggregated across poll cycles for the current ReplicaSet.
type WarningEntry struct {
	Message string
	Count   int
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

// CalculateRolloutStatus determines rollout status from deployment conditions and timing.
// Returns Complete if fully available, DeadlineExceeded if failed, DeadlineWarning if <25% time remains.
func CalculateRolloutStatus(deployment *appsv1.Deployment, newRS *appsv1.ReplicaSet) RolloutStatus {
	status := deployment.Status

	// Check for completion
	if isDeploymentComplete(status) {
		return StatusComplete
	}

	// Check for deadline exceeded
	if isDeploymentFailed(status) {
		return StatusDeadlineExceeded
	}

	// Check if approaching deadline (< 25% time remaining)
	// Use Progressing condition's LastUpdateTime - this resets whenever progress is made,
	// which matches Kubernetes' progress deadline behavior
	progressTime := getProgressUpdateTime(deployment)
	if progressTime != nil {
		deadline := getInt32OrDefault(deployment.Spec.ProgressDeadlineSeconds, DefaultProgressDeadlineSeconds)
		elapsed := time.Since(*progressTime)
		// Warn when more than 75% of deadline has elapsed (less than 25% remaining)
		warningThreshold := time.Duration(float64(deadline)*(1.0-DefaultDeadlineWarningThreshold)) * time.Second
		if elapsed > warningThreshold {
			return StatusDeadlineWarning
		}
	}

	return StatusProgressing
}

// hasCondition checks if a specific condition exists with given type and status.
// If reason is empty, only type/status checked. If provided, all three must match.
func hasCondition(status appsv1.DeploymentStatus, condType appsv1.DeploymentConditionType, condStatus corev1.ConditionStatus, reason string) bool {
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
