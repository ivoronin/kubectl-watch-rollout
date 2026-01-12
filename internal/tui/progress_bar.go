package tui

import (
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

// ProgressBar is the progress bar component.
type ProgressBar struct {
	bar     progress.Model
	width   int
	hasData bool
}

// NewProgressBar creates a new progress component.
func NewProgressBar() *ProgressBar {
	return &ProgressBar{
		bar: progress.New(
			progress.WithScaledGradient(string(ColorBlue), string(ColorGreen)),
			progress.WithFillCharacters('─', '─'),
			progress.WithoutPercentage(),
		),
	}
}

// SetWidth sets the component width.
func (m *ProgressBar) SetWidth(w int) { m.width = w }

// Update handles messages.
func (m *ProgressBar) Update(teaMsg tea.Msg) tea.Cmd {
	switch t := teaMsg.(type) {
	case SnapshotMsg:
		m.hasData = true
		return m.bar.SetPercent(float64(t.Snapshot.NewProgress))
	case progress.FrameMsg:
		model, cmd := m.bar.Update(t)
		if bar, ok := model.(progress.Model); ok {
			m.bar = bar
		}
		return cmd
	}
	return nil
}

// View renders the component.
func (m *ProgressBar) View() string {
	if !m.hasData {
		return ""
	}
	m.bar.Width = m.width
	return m.bar.View()
}
