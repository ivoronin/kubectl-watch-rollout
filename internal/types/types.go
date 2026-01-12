// Package types contains shared domain types used across monitor and tui packages.
package types //nolint:revive // types is a standard name for shared domain types

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
)

// RolloutStatus represents the current state of a deployment rollout.
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

// ReplicaSetState groups pod counts for a ReplicaSet at different lifecycle stages.
type ReplicaSetState struct {
	Current   int32
	Ready     int32
	Available int32
}

// EventCluster represents similar K8s events grouped together for display.
type EventCluster struct {
	Type          string    // K8s event type: "Warning" or "Normal"
	Reason        string    // K8s event reason (e.g., "FailedScheduling", "Unhealthy")
	Message       string    // Truncated representative message
	ExemplarCount int       // Total events matching this template
	LastSeen      time.Time // Most recent occurrence in cluster
}

// Symbol returns a visual symbol for display based on event Type.
func (e EventCluster) Symbol() string {
	if e.Type == corev1.EventTypeWarning {
		return "⚠"
	}
	return "ℹ"
}

// EventSummary is the result of event processing, ready for rendering.
type EventSummary struct {
	Clusters     []EventCluster // Event clusters ready for display
	IgnoredCount int            // Events filtered by ignore regex
}

// RolloutSnapshot represents a snapshot of the deployment rollout state.
// This is a pure domain DTO with no infrastructure dependencies.
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
	ProgressUpdateTime  *time.Time
	EstimatedCompletion *time.Time

	// Status and events
	Status RolloutStatus
	Events EventSummary
}

// View defines the interface for presenting rollout information.
type View interface {
	RenderSnapshot(snapshot *RolloutSnapshot)
	Shutdown()
	Done() <-chan struct{} // Signals view has exited (e.g., user pressed quit)
}

// FormatDuration formats duration with seconds precision.
func FormatDuration(d time.Duration) string {
	d = d.Round(time.Second)

	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		if m > 0 {
			if s > 0 {
				return fmt.Sprintf("%dh%dm%ds", h, m, s)
			}
			return fmt.Sprintf("%dh%dm", h, m)
		}
		if s > 0 {
			return fmt.Sprintf("%dh%ds", h, s)
		}
		return fmt.Sprintf("%dh", h)
	}

	if m > 0 {
		if s > 0 {
			return fmt.Sprintf("%dm%ds", m, s)
		}
		return fmt.Sprintf("%dm", m)
	}

	return fmt.Sprintf("%ds", s)
}
