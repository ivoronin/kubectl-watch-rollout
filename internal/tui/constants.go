// Package tui provides the terminal user interface components.
package tui

import "github.com/charmbracelet/lipgloss"

// Color constants used throughout the TUI.
const (
	ColorGray  = lipgloss.Color("#888888") // muted text, labels, headers
	ColorGreen = lipgloss.Color("#28D223") // complete, accents
	ColorBlue  = lipgloss.Color("#0493F8") // progressing, normal events
	ColorRed   = lipgloss.Color("#FF4444") // failed, warnings
)

// Shared table styles.
var (
	TableHeaderStyle = lipgloss.NewStyle().Foreground(ColorGray).Bold(true)
	TableLabelStyle  = lipgloss.NewStyle().Foreground(ColorGray)
)

// Layout styles.
var (
	// panelPaddingStyle is the standard padding for content panels (with top padding).
	panelPaddingStyle = lipgloss.NewStyle().PaddingTop(1).PaddingLeft(1).PaddingRight(1)

	// topPanelPaddingStyle is padding for top row panels (no top padding).
	topPanelPaddingStyle = lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1)

	// rowPaddingStyle is horizontal padding for single-row components (statusbar, progress).
	rowPaddingStyle = lipgloss.NewStyle().Padding(0, 1)

	// sectionTitleStyle is the base style for section titles with underline.
	// Use .Width(w).Render(title) to apply.
	sectionTitleStyle = lipgloss.NewStyle().
				Foreground(ColorGray).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(ColorGray)
)

// Layout constants.
const (
	// StatusbarH is the statusbar height (content only, progress bar acts as border).
	StatusbarH = 1
	// ProgressH is the progress bar height.
	ProgressH = 1

	// RolloutLabelColW is the rollout info label column width.
	RolloutLabelColW = 10
	// RolloutColPadding is the rollout info column padding.
	RolloutColPadding = 2

	// PodsStateColW is the pod stats column width.
	PodsStateColW = 9
	// PodsColPadding is the pod stats column padding.
	PodsColPadding = 2

	// EventsMinColW is the minimum events table column width.
	EventsMinColW = 10
	// EventsLastColW is the events "last seen" column width.
	EventsLastColW = 10
	// EventsColPadding is the events table column padding.
	EventsColPadding = 2
)
