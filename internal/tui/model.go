package tui

import (
	"github.com/charmbracelet/bubbles/key"
	bubbleprogress "github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model is the main bubbletea model for the TUI
type Model struct {
	width, height int
	hasData       bool
	quitting      bool

	spinner        spinner.Model
	keys           KeyMap
	rolloutInfo    *RolloutInfo
	progressBar    *ProgressBar
	podStats       *PodStats
	podsGrid       *PodsGrid
	eventsTable    *EventsTable
	eventsViewport viewport.Model
	statusbar      *Statusbar
}

// NewModel creates a new TUI model
func NewModel() Model {
	keys := DefaultKeyMap()
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorGreen)
	return Model{
		spinner:        s,
		keys:           keys,
		rolloutInfo:    NewRolloutInfo(),
		progressBar:    NewProgressBar(),
		podStats:       NewPodStats(),
		podsGrid:       NewPodsGrid(),
		eventsTable:    NewEventsTable(),
		eventsViewport: viewport.New(0, 0),
		statusbar:      NewStatusbar(keys),
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), m.spinner.Tick)
}

// Update implements tea.Model
func (m Model) Update(teaMsg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch t := teaMsg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = t.Width, t.Height

	case tea.KeyMsg:
		if key.Matches(t, m.keys.Quit) {
			m.quitting = true
			return m, tea.Quit
		}

	case SnapshotMsg:
		m.hasData = true
		cmds = append(cmds,
			m.rolloutInfo.Update(t),
			m.progressBar.Update(t),
			m.podStats.Update(t),
			m.podsGrid.Update(t),
			m.eventsTable.Update(t),
			m.statusbar.Update(t),
		)

	case spinner.TickMsg:
		if !m.hasData {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(teaMsg)
			cmds = append(cmds, cmd)
		}

	case TickMsg:
		cmds = append(cmds,
			m.rolloutInfo.Update(t),
			m.progressBar.Update(t),
			m.podStats.Update(t),
			m.eventsTable.Update(t),
			tickCmd(),
		)

	case bubbleprogress.FrameMsg:
		cmds = append(cmds, m.progressBar.Update(t))

	case QuitMsg:
		m.quitting = true
		return m, tea.Quit
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 || m.height == 0 {
		return ""
	}
	if !m.hasData {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			m.spinner.View()+"Loading...")
	}

	// Layout:
	// ┌─────────────────────────────┐
	// │         statusbar           │ StatusbarH
	// ┝━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┥ ProgressH (statusbar border)
	// │  rolloutInfo  │  podStats   │ topH (content + padding)
	// ├───────────────┴─────────────┤
	// │        eventsTable          │ eventsH (flex)
	// ├─────────────────────────────┤
	// │        podsGrid             │ podsGridH (content-driven)
	// └─────────────────────────────┘

	// Layout calculations using style frame sizes
	panelHFrame := panelPaddingStyle.GetHorizontalFrameSize()
	panelVFrame := panelPaddingStyle.GetVerticalFrameSize()
	contentWidth := m.width - panelHFrame

	// Render content-sized components
	rollout := m.rolloutInfo.View()
	stats := m.podStats.View()

	// Calculate top row dimensions (no top padding for top panels)
	statsW := lipgloss.Width(stats) + panelHFrame
	rolloutW := m.width - statsW
	topH := max(lipgloss.Height(rollout), lipgloss.Height(stats))

	// Render pods grid (content-driven height)
	m.podsGrid.SetWidth(contentWidth)
	podsGridRow := panelPaddingStyle.Width(m.width).Render(m.podsGrid.View())
	podsGridH := lipgloss.Height(podsGridRow)

	// Calculate flex height for events
	eventsH := m.height - StatusbarH - topH - podsGridH - ProgressH

	// Size remaining flex components
	m.progressBar.SetWidth(contentWidth) // with 1 char L/R padding
	m.eventsTable.SetWidth(contentWidth)
	m.statusbar.SetWidth(contentWidth)

	// Size and populate events viewport
	m.eventsViewport.Width = contentWidth
	m.eventsViewport.Height = max(0, eventsH-panelVFrame)
	m.eventsViewport.SetContent(m.eventsTable.View())

	// Compose layout
	statusRow := rowPaddingStyle.Render(m.statusbar.View())
	topRow := lipgloss.JoinHorizontal(lipgloss.Top,
		topPanelPaddingStyle.Width(rolloutW).Height(topH).Render(rollout),
		topPanelPaddingStyle.Width(statsW).Height(topH).Render(stats),
	)
	eventsRow := panelPaddingStyle.Width(m.width).Height(eventsH).Render(m.eventsViewport.View())
	progressRow := rowPaddingStyle.Render(m.progressBar.View()) // 1 char L/R padding

	return lipgloss.JoinVertical(lipgloss.Left,
		statusRow,
		progressRow,
		topRow,
		eventsRow,
		podsGridRow,
	)
}
