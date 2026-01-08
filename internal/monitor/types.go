package monitor

import (
	"errors"
	"regexp"
	"time"

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
	// DefaultSimilarityThreshold controls event clustering aggressiveness (0.0-1.0)
	// Higher values = less aggressive clustering (only near-identical messages grouped)
	// 0.85 keeps "Liveness" and "Readiness" probe failures separate
	DefaultSimilarityThreshold = 0.85
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
	UntilComplete       bool           // If true, exit after monitoring one rollout (default: false, continuous monitoring)
	LineMode            bool           // If true, use line-based output format (default: false, interactive mode)
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
		LineMode:            false, // Default: interactive mode
		IgnoreEvents:        nil,
	}
}

// RolloutStatus represents the current state of a deployment rollout.
// It is an enumeration with values defined as constants below.
type RolloutStatus int

const (
	// StatusProgressing indicates rollout is in progress
	StatusProgressing RolloutStatus = iota
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

// EventCluster represents similar K8s events grouped together for display.
// Used by both console and line renderers.
type EventCluster struct {
	Type           string    // K8s event type: "Warning" or "Normal"
	Reason         string    // K8s event reason (e.g., "FailedScheduling", "Unhealthy")
	Message        string    // Truncated representative message
	LookAlikeCount int       // Additional similar events (0 = standalone)
	LastSeen       time.Time // Most recent occurrence in cluster
}

// Symbol returns a visual symbol for display based on event Type.
func (e EventCluster) Symbol() string {
	if e.Type == corev1.EventTypeWarning {
		return "⚠"
	}
	return "ℹ"
}

// EventSummary is the result of event processing, ready for rendering.
// Contains clustered events plus metadata about filtering.
type EventSummary struct {
	Clusters     []EventCluster // Event clusters ready for display
	IgnoredCount int            // Events filtered by ignore regex
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
func CalculateRolloutStatus(deployment *appsv1.Deployment, newRS *appsv1.ReplicaSet) RolloutStatus {
	status := deployment.Status

	if isDeploymentComplete(status) {
		return StatusComplete
	}

	if isDeploymentFailed(status) {
		return StatusDeadlineExceeded
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
