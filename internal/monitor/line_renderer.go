// Package monitor provides Kubernetes deployment rollout monitoring functionality.
//
// This file contains line mode rendering logic for CI/CD-friendly output.
package monitor

import (
	"fmt"
	"io"
	"time"
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

// RenderSnapshot outputs a single timestamped status line followed by any events
func (r *LineRenderer) RenderSnapshot(snapshot *RolloutSnapshot) {
	statusLine := r.formatStatusLine(snapshot)
	fmt.Fprintln(r.output, statusLine)

	// Render events if any
	eventLines := r.formatEvents(snapshot.Events)
	for _, line := range eventLines {
		fmt.Fprintln(r.output, line)
	}

	// Add blank line for visual separation
	fmt.Fprintln(r.output)
}

// formatTimestamp returns compact local time (HH:MM:SS)
func (r *LineRenderer) formatTimestamp(t time.Time) string {
	return t.Local().Format("15:04:05")
}

// formatStatusLine generates the main status line
// Format: <timestamp> <symbol> [REPLICASET X] [ROLLOUT STATUS] [NEW X/Y] [OLD X/Y] [ETA/DUR]
func (r *LineRenderer) formatStatusLine(snapshot *RolloutSnapshot) string {
	symbol := r.formatSymbol(snapshot.Status)
	timestamp := r.formatTimestamp(snapshot.SnapshotTime)
	status := r.formatStatus(snapshot.Status)
	replicas := r.formatReplicaCounts(snapshot)
	metadata := r.formatMetadata(snapshot)

	return fmt.Sprintf("%s %s [REPLICASET %s] [ROLLOUT %s] %s %s", timestamp, symbol, snapshot.NewRSName, status, replicas, metadata)
}

// formatSymbol returns a visual symbol for the rollout status
func (r *LineRenderer) formatSymbol(status RolloutStatus) string {
	switch status {
	case StatusProgressing:
		return "▶"
	case StatusDeadlineExceeded:
		return "✗"
	case StatusComplete:
		return "✓"
	default:
		return "?"
	}
}

// formatStatus converts RolloutStatus enum to CAPS status word (matches K8s condition naming)
func (r *LineRenderer) formatStatus(status RolloutStatus) string {
	switch status {
	case StatusProgressing:
		return "PROGRESSING"
	case StatusDeadlineExceeded:
		return "DEADLINE-EXCEEDED"
	case StatusComplete:
		return "COMPLETE"
	default:
		return "UNKNOWN"
	}
}

// formatReplicaCounts formats replica counts for NEW and OLD ReplicaSets
// Format: [NEW X/Y] [OLD X/Y]
func (r *LineRenderer) formatReplicaCounts(snapshot *RolloutSnapshot) string {
	return fmt.Sprintf("[NEW %d/%d] [OLD %d/%d]",
		snapshot.NewRS.Available, snapshot.Desired,
		snapshot.OldRS.Available, snapshot.Desired)
}

// formatMetadata formats contextual metadata (ETA or DUR) in bracketed format
func (r *LineRenderer) formatMetadata(snapshot *RolloutSnapshot) string {
	// Show actual completion duration if rollout is done and we have ProgressUpdateTime
	if snapshot.Status.IsDone() {
		elapsed := snapshot.ProgressUpdateTime.Sub(snapshot.StartTime)
		return fmt.Sprintf("[DUR %s]", FormatDuration(elapsed))
	}

	// Show ETA if available
	if snapshot.EstimatedCompletion != nil {
		remaining := time.Until(*snapshot.EstimatedCompletion)
		return fmt.Sprintf("[ETA %s]", FormatDuration(remaining))
	}

	// No ETA available
	return "[ETA -]"
}

// formatEvents formats event lines with tree connector style for line mode.
func (r *LineRenderer) formatEvents(report EventSummary) []string {
	if len(report.Clusters) == 0 {
		return nil
	}

	var result []string
	for _, c := range report.Clusters {
		age := FormatDuration(time.Since(c.LastSeen)) + " ago"
		var line string
		if c.LookAlikeCount > 0 {
			line = fmt.Sprintf("         └─ %s %s: %s (+%d look-alike, last %s)",
				c.Symbol(), c.Reason, c.Message, c.LookAlikeCount, age)
		} else {
			line = fmt.Sprintf("         └─ %s %s: %s (last %s)",
				c.Symbol(), c.Reason, c.Message, age)
		}
		result = append(result, line)
	}

	if report.IgnoredCount > 0 {
		result = append(result, fmt.Sprintf("         └─ ... %d event(s) ignored", report.IgnoredCount))
	}

	return result
}
