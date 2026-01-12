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
	// Fixed row heights
	StatusbarH = 1 // statusbar height (content only, progress bar acts as border)
	ProgressH  = 1 // progress bar height

	// Rollout info
	RolloutLabelColW  = 10
	RolloutColPadding = 2

	// Pod stats
	PodsStateColW  = 9
	PodsColPadding = 2

	// Events table
	EventsMinColW    = 10
	EventsLastColW   = 10
	EventsColPadding = 2 // padding between columns (4 gaps for 5 columns)
)
