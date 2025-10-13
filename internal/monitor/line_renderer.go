// Package monitor provides Kubernetes deployment rollout monitoring functionality.
//
// This file contains line mode rendering logic for CI/CD-friendly output.
package monitor

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/duration"
)

// LineRenderer renders rollout status as single-line timestamped output
// suitable for CI/CD pipelines and log aggregation systems.
type LineRenderer struct {
	output io.Writer
	config Config
}

// NewLineRenderer creates a new line mode renderer
func NewLineRenderer(config Config, output io.Writer) *LineRenderer {
	return &LineRenderer{
		output: output,
		config: config,
	}
}

// RenderSnapshot outputs a single timestamped status line followed by any warnings
func (r *LineRenderer) RenderSnapshot(snapshot *RolloutSnapshot) {
	statusLine := r.formatStatusLine(snapshot)
	fmt.Fprintln(r.output, statusLine)

	// Render warnings if any
	warnings := r.formatWarnings(snapshot.Warnings)
	for _, warning := range warnings {
		fmt.Fprintln(r.output, warning)
	}
}

// formatTimestamp returns ISO 8601 UTC timestamp with milliseconds
func (r *LineRenderer) formatTimestamp(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

// formatStatusLine generates the main status line
// Format: <timestamp> replicaset/<rs-name> <status>: NEW:X/Y OLD:X/Y <metadata>
func (r *LineRenderer) formatStatusLine(snapshot *RolloutSnapshot) string {
	timestamp := r.formatTimestamp(snapshot.SnapshotTime)
	status := r.formatStatus(snapshot.Status)
	replicas := r.formatReplicaCounts(snapshot)
	metadata := r.formatMetadata(snapshot)

	return fmt.Sprintf("%s replicaset:%s status:%s %s %s", timestamp, snapshot.NewRSName, status, replicas, metadata)
}

// formatStatus converts RolloutStatus enum to string
func (r *LineRenderer) formatStatus(status RolloutStatus) string {
	switch status {
	case StatusProgressing:
		return "progressing"
	case StatusDeadlineExceeded:
		return "deadline-exceeded"
	case StatusComplete:
		return "complete"
	default:
		return "unknown"
	}
}

// formatReplicaCounts formats replica counts for NEW and OLD ReplicaSets
// Format: NEW:<available>/<desired> OLD:<available>/<desired>
func (r *LineRenderer) formatReplicaCounts(snapshot *RolloutSnapshot) string {
	newReplicas := fmt.Sprintf("new:%d/%d", snapshot.NewRS.Available, snapshot.Desired)
	oldReplicas := fmt.Sprintf("old:%d/%d", snapshot.OldRS.Available, snapshot.Desired)
	return fmt.Sprintf("%s %s", newReplicas, oldReplicas)
}

// formatMetadata formats contextual metadata (Started/ETA/Duration)
func (r *LineRenderer) formatMetadata(snapshot *RolloutSnapshot) string {
	// Show actual completion duration if rollout is done and we have ProgressUpdateTime
	if snapshot.Status.IsDone() {
		elapsed := snapshot.ProgressUpdateTime.Sub(snapshot.StartTime)
		return fmt.Sprintf("duration:%s", duration.ShortHumanDuration(elapsed))
	}

	// Show ETA if available
	if snapshot.EstimatedCompletion != nil {
		remaining := time.Until(*snapshot.EstimatedCompletion)
		return fmt.Sprintf("eta:%s", duration.ShortHumanDuration(remaining))
	}

	// No ETA available
	return "eta:-"
}

// formatWarnings formats warning lines with 4-space indentation
// Returns array of formatted warning strings
func (r *LineRenderer) formatWarnings(warnings []WarningEntry) []string {
	if len(warnings) == 0 {
		return nil
	}

	// Sort warnings by count (descending)
	sorted := make([]WarningEntry, len(warnings))
	copy(sorted, warnings)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Count > sorted[j].Count
	})

	// Truncate to max warnings
	maxWarnings := r.config.MaxWarnings
	var result []string

	for i, warning := range sorted {
		if i >= maxWarnings {
			remaining := len(sorted) - maxWarnings
			result = append(result, fmt.Sprintf("    ... %d more warning(s) not shown", remaining))
			break
		}

		// Sanitize message and format with prefix
		sanitizedMessage := r.sanitizeMessage(warning.Message)
		formatted := fmt.Sprintf("    WARNING: %s (%dx)", sanitizedMessage, warning.Count)
		result = append(result, formatted)
	}

	return result
}

// sanitizeMessage replaces newlines with spaces to maintain single-line format
func (r *LineRenderer) sanitizeMessage(msg string) string {
	// Replace newlines and carriage returns with space
	msg = strings.ReplaceAll(msg, "\n", " ")
	msg = strings.ReplaceAll(msg, "\r", " ")
	return msg
}
