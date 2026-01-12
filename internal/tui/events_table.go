package tui

import (
	"fmt"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/ivoronin/kubectl-watch-rollout/internal/types"
)

var (
	eventsWarningStyle = lipgloss.NewStyle().Foreground(ColorRed)
	eventsNormalStyle  = lipgloss.NewStyle().Foreground(ColorBlue)
)

// EventsTable is the events table component.
type EventsTable struct {
	width    int
	snapshot *types.RolloutSnapshot
}

// NewEventsTable creates a new events component.
func NewEventsTable() *EventsTable { return &EventsTable{} }

// SetWidth sets the component width.
func (m *EventsTable) SetWidth(w int) { m.width = w }

// Update handles messages.
func (m *EventsTable) Update(teaMsg tea.Msg) tea.Cmd {
	if t, ok := teaMsg.(SnapshotMsg); ok {
		m.snapshot = t.Snapshot
	}

	return nil
}

// View renders the component.
func (m *EventsTable) View() string {
	if m.snapshot == nil {
		return ""
	}

	title := m.buildEventsTitle()

	events := m.snapshot.Events.Clusters
	if len(events) == 0 {
		return title + "\n" + TableLabelStyle.Render("No events")
	}

	rows := formatEventRows(events)

	// Calculate column widths
	reasonW, similarW := EventsMinColW, EventsMinColW
	for _, r := range rows {
		reasonW = max(reasonW, len(r.reason))
		similarW = max(similarW, len(r.similar))
	}

	msgW := max(EventsMinColW, m.width-RolloutLabelColW-reasonW-similarW-EventsLastColW-4*EventsColPadding)
	colWidths := []int{
		RolloutLabelColW + EventsColPadding, reasonW + EventsColPadding,
		msgW + EventsColPadding, similarW + EventsColPadding, EventsLastColW,
	}

	// Build table rows
	tableRows := make([][]string, len(rows))
	for i, r := range rows {
		tableRows[i] = []string{r.eventType, r.reason, truncateStr(r.message, msgW), r.similar, r.last}
	}

	tbl := table.New().
		Headers("TYPE", "REASON", "MESSAGE", "EXEMPLARS", "LAST").
		Rows(tableRows...).
		BorderTop(false).BorderBottom(false).BorderLeft(false).BorderRight(false).
		BorderColumn(false).BorderRow(false).BorderHeader(false).
		StyleFunc(func(row, col int) lipgloss.Style {
			style := lipgloss.NewStyle().Width(colWidths[col])
			if col >= 3 {
				style = style.Align(lipgloss.Right)
			}

			if row == table.HeaderRow {
				return style.Inherit(TableHeaderStyle)
			}

			return style
		}).
		Render()

	return title + "\n" + tbl
}

// buildEventsTitle creates the section title with event stats.
func (m *EventsTable) buildEventsTitle() string {
	totalEvents := 0
	for _, c := range m.snapshot.Events.Clusters {
		totalEvents += c.ExemplarCount
	}

	stats := fmt.Sprintf("TOTAL %d  IGNORED %d", totalEvents, m.snapshot.Events.IgnoredCount)
	titleContent := "Events" + lipgloss.PlaceHorizontal(m.width-lipgloss.Width("Events"), lipgloss.Right, stats)

	return sectionTitleStyle.Width(m.width).Render(titleContent)
}

// eventRow holds formatted data for a single event row.
type eventRow struct {
	eventType, reason, message, similar, last string
}

// formatEventRows converts event clusters to formatted row data.
func formatEventRows(events []types.EventCluster) []eventRow {
	rows := make([]eventRow, len(events))

	for i, e := range events {
		rows[i] = eventRow{
			eventType: formatEventType(e.Type),
			reason:    e.Reason,
			message:   e.Message,
			similar:   formatExemplars(e.ExemplarCount),
			last:      types.FormatDuration(time.Since(e.LastSeen)),
		}
	}

	return rows
}

func truncateStr(s string, maxW int) string {
	if len(s) <= maxW {
		return s
	}

	if maxW <= 3 {
		return s[:maxW]
	}

	return s[:maxW-3] + "..."
}

func formatEventType(t string) string {
	if t == "Warning" {
		return eventsWarningStyle.Render("Warning")
	}

	return eventsNormalStyle.Render("Normal")
}

func formatExemplars(count int) string {
	return strconv.Itoa(count)
}
