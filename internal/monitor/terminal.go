package monitor

import (
	"fmt"
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
	isTTY bool
}

// NewTerminalController creates a new terminal controller
func NewTerminalController() *TerminalController {
	return &TerminalController{
		isTTY: term.IsTerminal(int(os.Stdout.Fd())),
	}
}

// ClearScreen clears the terminal screen
func (t *TerminalController) ClearScreen() {
	if t.isTTY {
		fmt.Print(ansiClearScreen)
	}
}

// SetProgress sets the terminal progress indicator
func (t *TerminalController) SetProgress(percent int) {
	if t.isTTY {
		fmt.Printf(ansiProgressFormat, percent)
	}
}

// ClearProgress clears the terminal progress indicator
func (t *TerminalController) ClearProgress() {
	if t.isTTY {
		fmt.Print(ansiClearProgress)
	}
}

// SetErrorState sets the terminal to error state
func (t *TerminalController) SetErrorState() {
	if t.isTTY {
		fmt.Print(ansiErrorState)
	}
}

// HideCursor hides the terminal cursor
func (t *TerminalController) HideCursor() {
	if t.isTTY {
		fmt.Print(ansiHideCursor)
	}
}

// ShowCursor shows the terminal cursor
func (t *TerminalController) ShowCursor() {
	if t.isTTY {
		fmt.Print(ansiShowCursor)
	}
}
