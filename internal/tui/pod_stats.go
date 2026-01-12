package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/ivoronin/kubectl-watch-rollout/internal/types"
)

// PodStats is the pods status table component.
type PodStats struct {
	snapshot *types.RolloutSnapshot
}

// NewPodStats creates a new pods component.
func NewPodStats() *PodStats { return &PodStats{} }

// Update handles messages.
func (m *PodStats) Update(teaMsg tea.Msg) tea.Cmd {
	if s, ok := teaMsg.(SnapshotMsg); ok {
		m.snapshot = s.Snapshot
	}
	return nil
}

// View renders the component.
func (m *PodStats) View() string {
	if m.snapshot == nil {
		return ""
	}
	s := m.snapshot
	totalAvl := s.OldRS.Available + s.NewRS.Available
	totalRdy := s.OldRS.Ready + s.NewRS.Ready
	totalCur := s.OldRS.Current + s.NewRS.Current

	maxVal := max(s.OldRS.Available, s.NewRS.Available, totalAvl,
		s.OldRS.Ready, s.NewRS.Ready, totalRdy,
		s.OldRS.Current, s.NewRS.Current, totalCur,
		s.Desired)
	numW := max(len(fmt.Sprintf("%d", maxVal)), len("TOT")) + PodsColPadding
	colWidths := []int{PodsStateColW, numW, numW, numW}

	rows := [][]string{
		{"AVAILABLE", fmt.Sprintf("%d", s.OldRS.Available), fmt.Sprintf("%d", s.NewRS.Available), fmt.Sprintf("%d", totalAvl)},
		{"READY", fmt.Sprintf("%d", s.OldRS.Ready), fmt.Sprintf("%d", s.NewRS.Ready), fmt.Sprintf("%d", totalRdy)},
		{"RUNNING", fmt.Sprintf("%d", s.OldRS.Current), fmt.Sprintf("%d", s.NewRS.Current), fmt.Sprintf("%d", totalCur)},
		{"DESIRED", "", "", fmt.Sprintf("%d", s.Desired)},
	}

	return table.New().
		Headers("POD STATE", "OLD", "NEW", "TOT").
		Rows(rows...).
		BorderTop(false).BorderBottom(false).BorderLeft(false).BorderRight(false).
		BorderColumn(false).BorderRow(false).BorderHeader(false).
		StyleFunc(func(row, col int) lipgloss.Style {
			style := lipgloss.NewStyle().Width(colWidths[col]).Align(lipgloss.Right)
			if row == table.HeaderRow {
				return style.Inherit(TableHeaderStyle)
			}
			if col == 0 {
				return style.Inherit(TableLabelStyle)
			}
			return style
		}).
		Render()
}
