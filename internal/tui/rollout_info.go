package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ivoronin/kubectl-watch-rollout/internal/types"
)

var (
	deploymentLabelStyle    = lipgloss.NewStyle().Foreground(ColorGray)
	deploymentProgressStyle = lipgloss.NewStyle().Foreground(ColorBlue).Bold(true)
	deploymentCompleteStyle = lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	deploymentFailedStyle   = lipgloss.NewStyle().Foreground(ColorRed).Bold(true)
)

// RolloutInfo is the deployment status component.
type RolloutInfo struct {
	snapshot *types.RolloutSnapshot
}

// NewRolloutInfo creates a new deployment component.
func NewRolloutInfo() *RolloutInfo { return &RolloutInfo{} }

// Update handles messages.
func (m *RolloutInfo) Update(teaMsg tea.Msg) tea.Cmd {
	if s, ok := teaMsg.(SnapshotMsg); ok {
		m.snapshot = s.Snapshot
	}
	return nil
}

// View renders the component.
func (m *RolloutInfo) View() string {
	if m.snapshot == nil {
		return ""
	}
	s := m.snapshot
	return strings.Join([]string{
		deploymentRow("Status", renderDeploymentStatus(s.Status)),
		deploymentRow("ReplicaSet", s.NewRSName),
		deploymentRow("Strategy", fmt.Sprintf("%s (Unavailable %s, Surge %s)", s.StrategyType, s.MaxUnavailable, s.MaxSurge)),
		deploymentRow("Started", fmt.Sprintf("%s (%s ago)", s.StartTime.Format("15:04:05"), types.FormatDuration(time.Since(s.StartTime)))),
		deploymentRow(deploymentETALabel(s), deploymentETAValue(s)),
	}, "\n")
}

func deploymentRow(label, value string) string {
	return deploymentLabelStyle.Render(label) + strings.Repeat(" ", RolloutLabelColW-len(label)+RolloutColPadding) + value
}

func renderDeploymentStatus(status types.RolloutStatus) string {
	switch status {
	case types.StatusComplete:
		return deploymentCompleteStyle.Render("Complete")
	case types.StatusDeadlineExceeded:
		return deploymentFailedStyle.Render("Deadline Exceeded")
	default:
		return deploymentProgressStyle.Render("Progressing")
	}
}

func deploymentETALabel(s *types.RolloutSnapshot) string {
	if s.Status == types.StatusComplete || s.Status == types.StatusDeadlineExceeded {
		return "Duration"
	}
	return "ETA"
}

func deploymentETAValue(s *types.RolloutSnapshot) string {
	switch {
	case s.Status == types.StatusComplete && s.ProgressUpdateTime != nil:
		return types.FormatDuration(s.ProgressUpdateTime.Sub(s.StartTime))
	case s.Status == types.StatusDeadlineExceeded:
		return "Failed"
	case s.EstimatedCompletion != nil:
		if rem := time.Until(*s.EstimatedCompletion); rem > 0 {
			return fmt.Sprintf("~%s remaining", types.FormatDuration(rem))
		}
		return "any moment..."
	default:
		return "Calculating..."
	}
}
