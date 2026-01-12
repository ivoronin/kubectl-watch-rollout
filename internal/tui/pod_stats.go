package tui

import (
	"strconv"

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
	numW := max(len(strconv.Itoa(int(maxVal))), len("TOT")) + PodsColPadding
	colWidths := []int{PodsStateColW, numW, numW, numW}

	i32 := func(v int32) string { return strconv.Itoa(int(v)) }
	rows := [][]string{
		{"AVAILABLE", i32(s.OldRS.Available), i32(s.NewRS.Available), i32(totalAvl)},
		{"READY", i32(s.OldRS.Ready), i32(s.NewRS.Ready), i32(totalRdy)},
		{"RUNNING", i32(s.OldRS.Current), i32(s.NewRS.Current), i32(totalCur)},
		{"DESIRED", "", "", i32(s.Desired)},
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
