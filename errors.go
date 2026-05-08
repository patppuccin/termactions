package termactions

import "errors"

// ErrInterrupted is returned when the user interrupts a prompt (e.g. Ctrl+C).
var ErrInterrupted = errors.New("prompt interrupted")

// ErrTerminalTooSmall is returned when the terminal dimensions are insufficient
// to render a component.
var ErrTerminalTooSmall = errors.New("terminal dimensions too small")

// ErrNoSelectionChoices is returned when a selection prompt is given no choices.
var ErrNoSelectionChoices = errors.New("no choices supplied for selection prompt")

// ErrInvalidSelectionBounds is returned when min count exceeds max count
// in a multi-select prompt configuration.
var ErrInvalidSelectionBounds = errors.New("min count must not exceed max count for multi select prompt")
