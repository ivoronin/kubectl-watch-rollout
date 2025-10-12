// Package monitor provides Kubernetes deployment rollout monitoring functionality.
//
// This file contains the view layer interfaces and implementations for rendering rollout status.
package monitor

import "io"

// View defines the interface for presenting rollout information
type View interface {
	// RenderSnapshot displays the current rollout state
	RenderSnapshot(snapshot *RolloutSnapshot)

	// Shutdown performs cleanup (cursor restoration, etc.)
	Shutdown()
}

// ConsoleView implements View for terminal output.
// Composes Renderer (formatting) and TerminalController (escape sequences).
type ConsoleView struct {
	renderer *Renderer
	terminal *TerminalController
}

// NewConsoleView creates a new console view
func NewConsoleView(config Config, writer io.Writer) *ConsoleView {
	terminal := NewTerminalController()
	terminal.HideCursor()

	return &ConsoleView{
		renderer: NewRenderer(config, writer),
		terminal: terminal,
	}
}

// RenderSnapshot displays the rollout snapshot with terminal control
func (v *ConsoleView) RenderSnapshot(snapshot *RolloutSnapshot) {
	v.terminal.ClearScreen()
	v.terminal.SetProgress(int(snapshot.NewProgress * 100))
	v.renderer.RenderSnapshot(snapshot)

	if snapshot.Status.IsFailed() {
		v.terminal.SetErrorState()
	} else if snapshot.Status.IsDone() {
		v.terminal.ClearProgress()
	}
}

// Shutdown restores terminal state
func (v *ConsoleView) Shutdown() {
	v.terminal.ShowCursor()
}
