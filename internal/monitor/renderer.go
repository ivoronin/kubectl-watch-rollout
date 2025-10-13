package monitor

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"
)

const (
	// timestampDisplayFormat is the standard time format used throughout the UI
	timestampDisplayFormat = "06-01-02 15:04:05"
)

// Renderer handles all output rendering and formatting.
// Converts RolloutSnapshot data into formatted text output.
type Renderer struct {
	config Config
	writer io.Writer
}

// NewRenderer creates a new renderer
func NewRenderer(config Config, writer io.Writer) *Renderer {
	if writer == nil {
		writer = os.Stdout
	}
	return &Renderer{
		config: config,
		writer: writer,
	}
}

// RenderSnapshot displays the current rollout snapshot
func (r *Renderer) RenderSnapshot(snapshot *RolloutSnapshot) {
	fmt.Fprintln(r.writer)
	r.renderHeader(snapshot)
	r.renderTiming(snapshot)
	fmt.Fprintln(r.writer)

	r.renderProgressBars(snapshot)
	r.renderLegend(snapshot)
	r.renderWarnings(snapshot)
	r.renderStatusLine(snapshot)
}

// renderHeader displays the rollout header with strategy information
func (r *Renderer) renderHeader(snapshot *RolloutSnapshot) {
	now := snapshot.SnapshotTime.Format(timestampDisplayFormat)
	rsName := snapshot.NewRSName

	switch snapshot.StrategyType {
	case "RollingUpdate":
		fmt.Fprintf(r.writer, " REPLICASET %s • RollingUpdate (U:%s S:%s) • %s\n",
			rsName, snapshot.MaxUnavailable, snapshot.MaxSurge, now)
	case "Recreate":
		fmt.Fprintf(r.writer, " REPLICASET %s • Recreate • %s\n", rsName, now)
	default:
		fmt.Fprintf(r.writer, " REPLICASET %s • %s • %s\n", rsName, snapshot.StrategyType, now)
	}
}

// renderTiming displays when the rollout started and ETA
func (r *Renderer) renderTiming(snapshot *RolloutSnapshot) {
	age := FormatDuration(snapshot.SnapshotTime.Sub(snapshot.StartTime))
	timestamp := snapshot.StartTime.Format(timestampDisplayFormat)

	fmt.Fprintln(r.writer)
	fmt.Fprintf(r.writer, " ⚐ STARTED: %s (%s ago)\n", timestamp, age)

	// Show actual completion time if rollout is done and we have ProgressUpdateTime
	if snapshot.Status.IsDone() && snapshot.ProgressUpdateTime != nil {
		finishTimestamp := snapshot.ProgressUpdateTime.Format(timestampDisplayFormat)
		elapsed := snapshot.ProgressUpdateTime.Sub(snapshot.StartTime)
		elapsedStr := FormatDuration(elapsed)
		fmt.Fprintf(r.writer, " ⚑ FINISHED: %s (%s elapsed)\n", finishTimestamp, elapsedStr)
	} else if snapshot.EstimatedCompletion != nil {
		etaStr := snapshot.EstimatedCompletion.Format(timestampDisplayFormat)
		remainingStr := FormatDuration(time.Until(*snapshot.EstimatedCompletion))
		fmt.Fprintf(r.writer, " ⚑ ETA:     %s (%s to go)\n", etaStr, remainingStr)
	} else {
		fmt.Fprintf(r.writer, " ⚑ ETA:     Calculating...\n")
	}
}

// renderProgressBars prints the NEW and OLD progress bars
func (r *Renderer) renderProgressBars(snapshot *RolloutSnapshot) {
	if snapshot.Desired == 0 {
		return
	}

	newBar := buildNewBar(r.config.ProgressBarWidth, snapshot.Desired, snapshot.NewRS.Current, snapshot.NewRS.Ready, snapshot.NewRS.Available)
	oldBar := buildOldBar(r.config.ProgressBarWidth, snapshot.Desired, snapshot.OldRS.Current)

	fmt.Fprintf(r.writer, " NEW %s %3.0f%%\n", newBar, snapshot.NewProgress*100)
	fmt.Fprintf(r.writer, " OLD %s %3.0f%%\n", oldBar, snapshot.OldProgress*100)
	fmt.Fprintln(r.writer)
}

// buildNewBar creates the NEW progress bar with overlays
func buildNewBar(width int, desired, current, ready, available int32) string {
	bar := make([]rune, width)
	for i := range bar {
		bar[i] = '·'
	}

	fill := func(count int32, char rune) {
		w := int(float64(count) / float64(desired) * float64(width))
		for i := 0; i < w && i < width; i++ {
			bar[i] = char
		}
	}

	fill(current, '░')
	fill(ready, '▒')
	fill(available, '█')

	return string(bar)
}

// buildOldBar creates the OLD progress bar
func buildOldBar(width int, desired, current int32) string {
	bar := make([]rune, width)
	for i := range bar {
		bar[i] = '·'
	}

	w := int(float64(current) / float64(desired) * float64(width))
	for i := 0; i < w && i < width; i++ {
		bar[i] = '◼'
	}

	return string(bar)
}

// renderLegend prints the pod count legend
func (r *Renderer) renderLegend(snapshot *RolloutSnapshot) {
	writer := tabwriter.NewWriter(r.writer, 0, 0, tablePaddingSpaces, ' ', 0)
	fmt.Fprintln(writer, "     \tOLD\tNEW\tTOTAL\tEXPLANATION")
	fmt.Fprintf(writer, "     █ AVAILABLE\t%d\t%d\t%d\tPods serving traffic\n",
		snapshot.OldRS.Available, snapshot.NewRS.Available, snapshot.OldRS.Available+snapshot.NewRS.Available)
	fmt.Fprintf(writer, "     ▒ READY\t%d\t%d\t%d\tPods passed readiness probe\n",
		snapshot.OldRS.Ready, snapshot.NewRS.Ready, snapshot.OldRS.Ready+snapshot.NewRS.Ready)
	fmt.Fprintf(writer, "     ░ CURRENT\t%d\t%d\t%d\tPods created\n",
		snapshot.OldRS.Current, snapshot.NewRS.Current, snapshot.OldRS.Current+snapshot.NewRS.Current)
	fmt.Fprintf(writer, "     · DESIRED\t\t\t%d\tTarget pod count\n",
		snapshot.Desired)
	writer.Flush()
	fmt.Fprintln(r.writer)
}

// renderWarnings prints warning events (max configurable)
func (r *Renderer) renderWarnings(snapshot *RolloutSnapshot) {
	if len(snapshot.Warnings) == 0 && snapshot.IgnoredWarningsCount == 0 {
		return
	}

	fmt.Fprintln(r.writer, " WARNINGS:")

	limit := len(snapshot.Warnings)
	if limit > r.config.MaxWarnings {
		limit = r.config.MaxWarnings
	}

	for i := 0; i < limit; i++ {
		fmt.Fprintf(r.writer, "  ⚠ %s (%dx)\n", snapshot.Warnings[i].Message, snapshot.Warnings[i].Count)
	}

	if len(snapshot.Warnings) > r.config.MaxWarnings {
		fmt.Fprintf(r.writer, "    ... %d more warning(s) not shown\n", len(snapshot.Warnings)-r.config.MaxWarnings)
	}

	if snapshot.IgnoredWarningsCount > 0 {
		fmt.Fprintf(r.writer, "    ... %d warning(s) ignored\n", snapshot.IgnoredWarningsCount)
	}
}

// renderStatusLine displays the current rollout status
func (r *Renderer) renderStatusLine(snapshot *RolloutSnapshot) {
	fmt.Fprintln(r.writer)
	switch snapshot.Status {
	case StatusProgressing:
		fmt.Fprintln(r.writer, " STATUS: → Progressing")
	case StatusDeadlineExceeded:
		fmt.Fprintln(r.writer, " STATUS: ✗ Deadline Exceeded")
	case StatusComplete:
		fmt.Fprintln(r.writer, " STATUS: ✓ Complete")
	}
}
