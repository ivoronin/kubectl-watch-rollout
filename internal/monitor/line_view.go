// Package monitor provides Kubernetes deployment rollout monitoring functionality.
//
// This file contains the LineView implementation for CI/CD-friendly output.
package monitor

import (
	"io"

	"github.com/ivoronin/kubectl-watch-rollout/internal/types"
)

// LineView implements View for line-based output suitable for CI/CD pipelines.
// Uses LineRenderer for formatting and outputs timestamped status lines.
type LineView struct {
	renderer *LineRenderer
}

// NewLineView creates a new line mode view
func NewLineView(config Config, writer io.Writer) *LineView {
	return &LineView{
		renderer: NewLineRenderer(config, writer),
	}
}

// RenderSnapshot displays the rollout snapshot as timestamped line output
func (v *LineView) RenderSnapshot(snapshot *types.RolloutSnapshot) {
	v.renderer.RenderSnapshot(snapshot)
}

// Shutdown performs cleanup (no-op for line mode - no terminal state to restore)
func (v *LineView) Shutdown() {
	// No cleanup needed for line mode
}

// Done implements types.View
// Returns nil - line mode is non-interactive, exits via Ctrl+C only
func (v *LineView) Done() <-chan struct{} {
	return nil
}
