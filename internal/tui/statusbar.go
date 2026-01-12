package tui

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	statusbarTextStyle = lipgloss.NewStyle().Foreground(ColorGray)
	statusbarNameStyle = lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
)

// Statusbar is the status bar component.
type Statusbar struct {
	width          int
	deploymentName string
	help           help.Model
	keys           help.KeyMap
}

// NewStatusbar creates a new status bar component.
func NewStatusbar(keys help.KeyMap) *Statusbar {
	h := help.New()
	h.ShowAll = false
	return &Statusbar{help: h, keys: keys}
}

// SetWidth sets the component width.
func (m *Statusbar) SetWidth(w int) { m.width = w }

// Update handles messages.
func (m *Statusbar) Update(teaMsg tea.Msg) tea.Cmd {
	switch t := teaMsg.(type) {
	case SnapshotMsg:
		m.deploymentName = t.Snapshot.DeploymentName
	}
	return nil
}

// View renders the component.
func (m *Statusbar) View() string {
	left := statusbarTextStyle.Render("Watching rollout for deployment ") + statusbarNameStyle.Render(m.deploymentName)
	right := statusbarTextStyle.Render(m.help.View(m.keys))
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	content := left + lipgloss.NewStyle().Width(gap).Render("") + right
	return lipgloss.NewStyle().
		Width(m.width).
		Render(content)
}
