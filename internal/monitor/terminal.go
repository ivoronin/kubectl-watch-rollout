package monitor

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

const (
	// ANSI escape sequences for terminal control
	ansiClearScreen    = "\033[H\033[2J"      // Move cursor to home + clear screen
	ansiHideCursor     = "\033[?25l"          // Hide cursor
	ansiShowCursor     = "\033[?25h"          // Show cursor (restore visibility)
	ansiProgressFormat = "\033]9;4;1;%d\007"  // Windows Terminal progress: state=1 (normal), percent=%d
	ansiClearProgress  = "\033]9;4;0;\007"    // Windows Terminal progress: state=0 (clear)
	ansiErrorState     = "\033]9;4;2;100\007" // Windows Terminal progress: state=2 (error), 100%
)

// TerminalController handles terminal-specific operations via ANSI escape sequences.
// Auto-detects TTY and only emits control sequences when appropriate.
type TerminalController struct {
	writer io.Writer
	isTTY  bool
}

// NewTerminalController creates a new terminal controller with the specified writer.
// If writer is nil, defaults to os.Stdout.
func NewTerminalController(writer io.Writer) *TerminalController {
	if writer == nil {
		writer = os.Stdout
	}

	// Detect if writer is a TTY (only works for *os.File)
	isTTY := false
	if f, ok := writer.(*os.File); ok {
		isTTY = term.IsTerminal(int(f.Fd()))
	}

	return &TerminalController{
		writer: writer,
		isTTY:  isTTY,
	}
}

// ClearScreen clears the terminal screen
func (t *TerminalController) ClearScreen() {
	if t.isTTY {
		fmt.Fprint(t.writer, ansiClearScreen)
	}
}

// SetProgress sets the terminal progress indicator
func (t *TerminalController) SetProgress(percent int) {
	if t.isTTY {
		fmt.Fprintf(t.writer, ansiProgressFormat, percent)
	}
}

// ClearProgress clears the terminal progress indicator
func (t *TerminalController) ClearProgress() {
	if t.isTTY {
		fmt.Fprint(t.writer, ansiClearProgress)
	}
}

// SetErrorState sets the terminal to error state
func (t *TerminalController) SetErrorState() {
	if t.isTTY {
		fmt.Fprint(t.writer, ansiErrorState)
	}
}

// HideCursor hides the terminal cursor
func (t *TerminalController) HideCursor() {
	if t.isTTY {
		fmt.Fprint(t.writer, ansiHideCursor)
	}
}

// ShowCursor shows the terminal cursor
func (t *TerminalController) ShowCursor() {
	if t.isTTY {
		fmt.Fprint(t.writer, ansiShowCursor)
	}
}
