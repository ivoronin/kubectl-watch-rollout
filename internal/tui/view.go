package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ivoronin/kubectl-watch-rollout/internal/types"
)

// View implements types.View using bubbletea
type View struct {
	program *tea.Program
	done    chan struct{}
}

// NewView creates and starts the TUI
func NewView() *View {
	done := make(chan struct{})

	model := NewModel()
	program := tea.NewProgram(model, tea.WithAltScreen())

	view := &View{
		program: program,
		done:    done,
	}

	// Start TUI in background goroutine
	go func() {
		defer close(done)

		_, _ = program.Run()
	}()

	return view
}

// RenderSnapshot implements types.View
func (v *View) RenderSnapshot(snapshot *types.RolloutSnapshot) {
	v.program.Send(SnapshotMsg{Snapshot: snapshot})
}

// Shutdown implements types.View
func (v *View) Shutdown() {
	v.program.Quit()
	<-v.done
}

// Done implements types.View
func (v *View) Done() <-chan struct{} {
	return v.done
}
