package termactions

import (
	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
)

// stdOutput is the colorable stdout used by all Termactions components.
// On Windows, this ensures ANSI escape sequences render correctly.
var stdOutput = colorable.NewColorableStdout()

// StyleMap defines the visual appearance of all Termactions TUI components.
// Every field is a [*color.Color] from the fatih/color package — assign
// any value constructed with [color.New] to override a specific style.
//
// As a good practice, always obtain a StyleMap via [NewStyles] — do not
// construct it directly. Unset fields will be unstyled at runtime.
//
//	styles := termactions.NewStyles()
//	styles.InputPrefix = color.New(color.FgMagenta)
type StyleMap struct {
	// Text-based message styles.
	MsgSuccessPrefix *color.Color
	MsgSuccessLabel  *color.Color
	MsgDebugPrefix   *color.Color
	MsgDebugLabel    *color.Color
	MsgInfoPrefix    *color.Color
	MsgInfoLabel     *color.Color
	MsgWarnPrefix    *color.Color
	MsgWarnLabel     *color.Color
	MsgErrorPrefix   *color.Color
	MsgErrorLabel    *color.Color
	MsgNeutral       *color.Color
	MsgMuted         *color.Color

	// Input prompt styles.
	InputPrefix         *color.Color
	InputLabel          *color.Color
	InputPlaceholder    *color.Color
	InputText           *color.Color
	InputValidationFail *color.Color
	InputHelp           *color.Color

	// Confirmation prompt styles.
	ConfirmationPrefix *color.Color
	ConfirmationLabel  *color.Color
	ConfirmationHelp   *color.Color

	// Selection prompt styles.
	SelectionPrefix             *color.Color
	SelectionLabel              *color.Color
	SelectionHelp               *color.Color
	SelectionSearchLabel        *color.Color
	SelectionSearchText         *color.Color
	SelectionSearchHint         *color.Color
	SelectionValidationFail     *color.Color
	SelectionItemNormalMarker   *color.Color
	SelectionItemNormalLabel    *color.Color
	SelectionItemCurrentMarker  *color.Color
	SelectionItemCurrentLabel   *color.Color
	SelectionItemSelectedMarker *color.Color
	SelectionItemSelectedLabel  *color.Color

	// Spinner styles.
	SpinnerPrefix *color.Color
	SpinnerLabel  *color.Color

	// Progress bar styles.
	ProgressPrefix     *color.Color
	ProgressLabel      *color.Color
	ProgressBarPad     *color.Color
	ProgressBarDone    *color.Color
	ProgressBarPending *color.Color
	ProgressBarStatus  *color.Color
}

// NewStyles returns a [StyleMap] with sensible default colors.
//
// The palette uses sharp and distinctive colors with semantic states
// such as green for success, yellow for warnings, red for errors,
// blue for info, and dark gray for muted/dimmed elements.
func NewStyles() *StyleMap {
	return &StyleMap{
		// Text-based message
		MsgSuccessPrefix: color.New(color.FgGreen),
		MsgSuccessLabel:  color.New(color.Reset),
		MsgDebugPrefix:   color.New(color.FgHiBlack),
		MsgDebugLabel:    color.New(color.Reset),
		MsgInfoPrefix:    color.New(color.FgBlue),
		MsgInfoLabel:     color.New(color.Reset),
		MsgWarnPrefix:    color.New(color.FgYellow),
		MsgWarnLabel:     color.New(color.Reset),
		MsgErrorPrefix:   color.New(color.FgRed),
		MsgErrorLabel:    color.New(color.Reset),
		MsgNeutral:       color.New(color.Reset),
		MsgMuted:         color.New(color.FgHiBlack),

		// Input prompts
		InputPrefix:         color.New(color.FgYellow),
		InputLabel:          color.New(color.Reset),
		InputPlaceholder:    color.New(color.FgHiBlack),
		InputText:           color.New(color.Reset),
		InputValidationFail: color.New(color.FgRed),
		InputHelp:           color.New(color.FgHiBlack),

		// Confirmation prompts
		ConfirmationPrefix: color.New(color.FgYellow),
		ConfirmationLabel:  color.New(color.Reset),
		ConfirmationHelp:   color.New(color.FgHiBlack),

		// Selection prompts
		SelectionPrefix:             color.New(color.FgYellow),
		SelectionLabel:              color.New(color.Reset),
		SelectionHelp:               color.New(color.FgHiBlack),
		SelectionSearchLabel:        color.New(color.FgYellow),
		SelectionSearchText:         color.New(color.Reset),
		SelectionSearchHint:         color.New(color.FgHiBlack),
		SelectionValidationFail:     color.New(color.FgRed),
		SelectionItemNormalMarker:   color.New(color.Reset),
		SelectionItemNormalLabel:    color.New(color.Reset),
		SelectionItemCurrentMarker:  color.New(color.FgYellow),
		SelectionItemCurrentLabel:   color.New(color.FgHiYellow),
		SelectionItemSelectedMarker: color.New(color.FgGreen),
		SelectionItemSelectedLabel:  color.New(color.FgGreen),

		// Spinners
		SpinnerPrefix: color.New(color.FgYellow),
		SpinnerLabel:  color.New(color.Reset),

		// Progress bars
		ProgressPrefix:     color.New(color.FgYellow),
		ProgressLabel:      color.New(color.Reset),
		ProgressBarPad:     color.New(color.FgYellow),
		ProgressBarDone:    color.New(color.FgYellow),
		ProgressBarPending: color.New(color.FgHiBlack),
		ProgressBarStatus:  color.New(color.Reset),
	}
}
