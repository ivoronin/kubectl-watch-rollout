package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ivoronin/kubectl-watch-rollout/internal/types"
)

const (
	symbolAvailable = "■"
	symbolReady     = "◧"
	symbolCurrent   = "□"
)

var (
	newPodStyle = lipgloss.NewStyle().Foreground(ColorGreen)
	oldPodStyle = lipgloss.NewStyle().Foreground(ColorGray)
)

// PodsGrid displays individual pod status as symbols in a grid.
type PodsGrid struct {
	width    int
	snapshot *types.RolloutSnapshot
}

// NewPodsGrid creates a new pods grid component.
func NewPodsGrid() *PodsGrid { return &PodsGrid{} }

// SetWidth sets the component width.
func (m *PodsGrid) SetWidth(w int) { m.width = w }

// Update handles messages.
func (m *PodsGrid) Update(teaMsg tea.Msg) tea.Cmd {
	if s, ok := teaMsg.(SnapshotMsg); ok {
		m.snapshot = s.Snapshot
	}
	return nil
}

// View renders the component.
func (m *PodsGrid) View() string {
	if m.snapshot == nil || m.width <= 0 {
		return ""
	}

	// Build title with legend on right
	left := "Pods"
	legend := symbolAvailable + " AVAILABLE  " + symbolReady + " READY  " + symbolCurrent + " RUNNING"
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(legend)
	if gap < 1 {
		gap = 1
	}
	titleLine := left + strings.Repeat(" ", gap) + legend
	title := sectionTitleStyle.Width(m.width).Render(titleLine)

	// Calculate pod counts (Current >= Ready >= Available)
	newAvail := int(m.snapshot.NewRS.Available)
	newReady := int(m.snapshot.NewRS.Ready) - newAvail
	newCurrent := int(m.snapshot.NewRS.Current) - int(m.snapshot.NewRS.Ready)
	oldAvail := int(m.snapshot.OldRS.Available)
	oldReady := int(m.snapshot.OldRS.Ready) - oldAvail
	oldCurrent := int(m.snapshot.OldRS.Current) - int(m.snapshot.OldRS.Ready)

	// Build symbol sequence: NEW (avail, ready, current) then OLD (avail, ready, current)
	var symbols []string
	symbols = append(symbols, repeat(newPodStyle.Render(symbolAvailable), newAvail)...)
	symbols = append(symbols, repeat(newPodStyle.Render(symbolReady), newReady)...)
	symbols = append(symbols, repeat(newPodStyle.Render(symbolCurrent), newCurrent)...)
	symbols = append(symbols, repeat(oldPodStyle.Render(symbolAvailable), oldAvail)...)
	symbols = append(symbols, repeat(oldPodStyle.Render(symbolReady), oldReady)...)
	symbols = append(symbols, repeat(oldPodStyle.Render(symbolCurrent), oldCurrent)...)

	if len(symbols) == 0 {
		return title
	}

	// Wrap symbols at width (each symbol is 1 char + 1 space, except last)
	symbolsPerLine := (m.width + 1) / 2
	if symbolsPerLine <= 0 {
		symbolsPerLine = 1
	}

	var lines []string
	for i := 0; i < len(symbols); i += symbolsPerLine {
		end := i + symbolsPerLine
		if end > len(symbols) {
			end = len(symbols)
		}
		lines = append(lines, strings.Join(symbols[i:end], " "))
	}

	return title + "\n" + strings.Join(lines, "\n")
}

func repeat(s string, n int) []string {
	if n <= 0 {
		return nil
	}
	result := make([]string, n)
	for i := range result {
		result[i] = s
	}
	return result
}
