// Package tui provides a bubbletea-based terminal UI for monitoring rollouts.
package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ivoronin/kubectl-watch-rollout/internal/types"
)

// External messages (from controller to TUI)

// SnapshotMsg delivers new rollout data from the controller.
type SnapshotMsg struct {
	Snapshot *types.RolloutSnapshot
}

// TickMsg triggers periodic UI updates (for "X ago" times).
type TickMsg time.Time

// QuitMsg signals the TUI should exit.
type QuitMsg struct{}

// tickCmd returns a command for periodic refresh (every 1s for time displays).
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}
