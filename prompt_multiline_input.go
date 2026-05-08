package termactions

import (
	"bufio"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
)

// multilineText renders an interactive multi-line text prompt.
// Construct one with [MultilineText].
type multilineText struct {
	cfg          Config
	prefix       string
	label        string
	placeholder  string
	defaultValue string
	validator    func(string) (string, bool)
}

// MultilineText returns a builder for an interactive multi-line text prompt.
//
//	desc, err := termactions.MultilineText().WithLabel("Description").Render()
//	if errors.Is(err, termactions.ErrInterrupted) { ... }
func MultilineText() *multilineText {
	return &multilineText{
		cfg:   pkgConfig,
		label: "Enter value",
	}
}

// WithStyles overrides the [StyleMap] for this prompt.
func (a *multilineText) WithStyles(s *StyleMap) *multilineText {
	a.cfg.Styles = s
	return a
}

// WithPrefix overrides the default prompt prefix symbol.
func (a *multilineText) WithPrefix(p string) *multilineText {
	a.prefix = p
	return a
}

// WithLabel sets the prompt label shown to the user.
func (a *multilineText) WithLabel(l string) *multilineText {
	a.label = l
	return a
}

// WithPlaceholder sets placeholder text shown when the input is empty.
func (a *multilineText) WithPlaceholder(p string) *multilineText {
	a.placeholder = p
	return a
}

// WithDefaultValue sets a default value used when the user submits empty input.
func (a *multilineText) WithDefaultValue(v string) *multilineText {
	a.defaultValue = v
	return a
}

// WithValidator sets a validation function called on submit.
// Returns a message and false to block submission, or a message and true to allow.
func (a *multilineText) WithValidator(fn func(string) (string, bool)) *multilineText {
	a.validator = fn
	return a
}

// Render displays the interactive prompt and blocks until the user submits or
// cancels. Returns the entered string, or [ErrInterrupted] if Ctrl+C is pressed.
//
// In accessible mode, input is collected line-by-line until a blank line is entered.
// Validation is checked on submit and the prompt reprints until satisfied.
func (a *multilineText) Render() (string, error) {
	if a.cfg.Accessible {
		return a.renderAccessible()
	}
	return a.renderInteractive()
}

// renderAccessible collects multiline input without cursor magic.
// Lines are read until the user enters a blank line to submit.
// Validation is checked on submit and the prompt reprints on failure.
func (a *multilineText) renderAccessible() (string, error) {
	prefix := pick(a.prefix, "(?)")
	promptLine := safeStyle(a.cfg.Styles.InputPrefix).Sprint(prefix) + " " +
		safeStyle(a.cfg.Styles.InputLabel).Sprint(a.label)

	placeholder := ""
	if a.placeholder != "" {
		placeholder = safeStyle(a.cfg.Styles.InputPlaceholder).Sprint(a.placeholder)
	}

	for {
		stdOutput.Write([]byte(promptLine + "\n"))
		if placeholder != "" {
			stdOutput.Write([]byte(placeholder + "\n"))
		}
		stdOutput.Write([]byte(safeStyle(a.cfg.Styles.InputHelp).Sprint("(enter a blank line to submit)") + "\n"))

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

		type readResult struct {
			line string
			err  error
		}

		var lines []string
		reader := bufio.NewReader(os.Stdin)

		for {
			ch := make(chan readResult, 1)
			go func() {
				line, err := reader.ReadString('\n')
				ch <- readResult{line, err}
			}()

			select {
			case <-sigCh:
				signal.Stop(sigCh)
				return "", ErrInterrupted
			case r := <-ch:
				if r.err != nil {
					signal.Stop(sigCh)
					if isInterrupt(r.err) {
						return "", ErrInterrupted
					}
					return "", r.err
				}
				trimmed := strings.TrimRight(r.line, "\r\n")
				if trimmed == "" {
					goto submit
				}
				lines = append(lines, trimmed)
			}
		}

	submit:
		signal.Stop(sigCh)
		result := strings.Join(lines, "\n")

		if a.validator != nil {
			msg, ok := a.validator(result)
			if !ok {
				stdOutput.Write([]byte(safeStyle(a.cfg.Styles.InputValidationFail).Sprint(msg) + "\n\n"))
				continue
			}
		}

		if result == "" && a.defaultValue != "" {
			result = a.defaultValue
		}

		return result, nil
	}
}

// renderInteractive renders the animated multi-line prompt with live redraws.
//
// Layout:
//
//	(?) Label:
//	<blank>
//	placeholder or typed lines
//	<blank>
//	validation line
//	help line
func (a *multilineText) renderInteractive() (string, error) {
	const (
		minTermWidth  = 42
		minTermHeight = 8
	)

	var (
		cursorRow     = 0 // zero-based row of cursor within the full frame
		prefix        = pick(a.prefix, "(?)")
		lines         = [][]rune{{}} // at least one line
		lineIdx       = 0            // which line the cursor is on
		colIdx        = 0            // cursor column within the current line
		interrupted   = false
		receivedInput = false
		firstRender   = true
	)

	// Guard against small terminal dimensions
	if w, h, err := termSize(); err != nil || w < minTermWidth || h < minTermHeight {
		return "", ErrTerminalTooSmall
	}

	// Build static segments
	promptLine := safeStyle(a.cfg.Styles.InputPrefix).Sprint(prefix) + " " +
		safeStyle(a.cfg.Styles.InputLabel).Sprint(a.label) + ":"
	helpLine := safeStyle(a.cfg.Styles.InputHelp).Sprint("ctrl+d to confirm  •  ctrl+c to cancel")

	// joinLines returns the full text content from all lines.
	joinLines := func() string {
		parts := make([]string, len(lines))
		for idx, l := range lines {
			parts[idx] = string(l)
		}
		return strings.Join(parts, "\n")
	}

	// buildContentLines returns the display lines for the text area.
	buildContentLines := func() []string {
		if len(lines) == 1 && len(lines[0]) == 0 {
			// Empty — show placeholder or default
			if a.defaultValue != "" && a.placeholder != "" {
				return []string{safeStyle(a.cfg.Styles.InputPlaceholder).Sprint(a.placeholder + " (default: " + a.defaultValue + ")")}
			} else if a.defaultValue != "" {
				return []string{safeStyle(a.cfg.Styles.InputPlaceholder).Sprint(a.defaultValue)}
			} else if a.placeholder != "" {
				return []string{safeStyle(a.cfg.Styles.InputPlaceholder).Sprint(a.placeholder)}
			}
			return []string{""}
		}
		result := make([]string, len(lines))
		for idx, l := range lines {
			result[idx] = safeStyle(a.cfg.Styles.InputText).Sprint(string(l))
		}
		return result
	}

	redraw := func(validationMsg string) {
		termW, termH, _ := termSize()

		// Build the content lines
		contentLines := buildContentLines()

		// Only show validation after user has started typing
		validationLine := ""
		if a.validator != nil && receivedInput && validationMsg != "" {
			validationLine = safeStyle(a.cfg.Styles.InputValidationFail).Sprint(validationMsg)
		}

		// Frame: prompt, blank, content..., blank, validation, help
		frameLines := []string{promptLine, ""}
		frameLines = append(frameLines, contentLines...)
		frameLines = append(frameLines, "", validationLine, helpLine)
		frameHeight := totalPhysicalLines(frameLines, termW)

		// Move cursor back to row 0 of the frame
		if !firstRender {
			ansiCursorUp(cursorRow)
		}

		if termH < frameHeight || termW < minTermWidth || termH < minTermHeight {
			stdOutput.Write([]byte(
				"\r" + ansiClearScreen +
					safeStyle(a.cfg.Styles.InputValidationFail).Sprint("terminal too small to render content"),
			))
			cursorRow = 0
			firstRender = true
			return
		}

		// Write the full frame
		stdOutput.Write([]byte(ansiHideCursor))

		var b strings.Builder
		for idx, line := range frameLines {
			if idx == len(frameLines)-1 {
				b.WriteString("\r" + line + ansiClearLine)
			} else {
				b.WriteString("\r" + line + ansiClearLine + "\n")
			}
		}
		b.WriteString(ansiClearScreen)
		stdOutput.Write([]byte(b.String()))

		// Move from last frame line back to row 0
		ansiCursorUp(frameHeight - 1)

		// Position cursor by reprinting content up to the cursor point.
		isEmpty := len(lines) == 1 && len(lines[0]) == 0

		if isEmpty {
			// Cursor belongs on the empty content line (after prompt + blank)
			reprint := promptLine + "\n\n"
			stdOutput.Write([]byte("\r" + reprint))
			cursorRow = physicalLines(stripAnsi(promptLine), termW) - 1 + 2 // prompt rows + blank + content row
		} else {
			// Reprint prompt + blank + content lines up to and including cursor line
			var reprint strings.Builder
			reprint.WriteString(promptLine + "\n\n")
			for idx := 0; idx <= lineIdx; idx++ {
				if idx == lineIdx {
					// Only up to cursor column on the cursor line
					before := string(lines[idx][:colIdx])
					reprint.WriteString(safeStyle(a.cfg.Styles.InputText).Sprint(before))
				} else {
					reprint.WriteString(safeStyle(a.cfg.Styles.InputText).Sprint(string(lines[idx])))
					reprint.WriteString("\n")
				}
			}
			stdOutput.Write([]byte("\r" + reprint.String()))

			// Calculate cursor row: prompt physical rows + 1 blank + content rows up to cursor
			plainPromptRows := physicalLines(stripAnsi(promptLine), termW)
			contentRowsBefore := 0
			for idx := 0; idx < lineIdx; idx++ {
				contentRowsBefore += physicalLines(string(lines[idx]), termW)
			}
			// Current line up to cursor
			cursorLineUpTo := string(lines[lineIdx][:colIdx])
			if cursorLineUpTo == "" {
				contentRowsBefore += 0 // cursor at start of line, same row
			} else {
				contentRowsBefore += physicalLines(cursorLineUpTo, termW) - 1
			}
			cursorRow = plainPromptRows + 1 + contentRowsBefore // +1 for blank line
		}

		stdOutput.Write([]byte(ansiShowCursor))
		firstRender = false
	}

	// Prep for render, hide cursor, defer cleanup
	stdOutput.Write([]byte("\r" + ansiHideCursor))
	defer func() {
		ansiCursorUp(cursorRow)
		stdOutput.Write([]byte("\r" + ansiClearScreen + ansiReset + ansiShowCursor))
	}()

	// Initial render
	redraw("")

	err := listenKeys(func(ev keyEvent) (stop bool) {
		switch ev.code {
		case keyCtrlC:
			interrupted = true
			return true

		case keyCtrlD:
			// Submit
			if a.validator != nil {
				msg, ok := a.validator(joinLines())
				if !ok {
					receivedInput = true
					redraw(msg)
					return false
				}
			}
			if len(lines) == 1 && len(lines[0]) == 0 && a.defaultValue != "" {
				lines = [][]rune{[]rune(a.defaultValue)}
			}
			receivedInput = true
			return true

		case keyEnter:
			// Insert a new line
			tail := append([]rune{}, lines[lineIdx][colIdx:]...)
			lines[lineIdx] = lines[lineIdx][:colIdx]
			lines = slices.Insert(lines, lineIdx+1, tail)
			lineIdx++
			colIdx = 0

		case keyLeft:
			if colIdx > 0 {
				colIdx--
			} else if lineIdx > 0 {
				lineIdx--
				colIdx = len(lines[lineIdx])
			}

		case keyRight:
			if colIdx < len(lines[lineIdx]) {
				colIdx++
			} else if lineIdx < len(lines)-1 {
				lineIdx++
				colIdx = 0
			}

		case keyUp:
			if lineIdx > 0 {
				lineIdx--
				if colIdx > len(lines[lineIdx]) {
					colIdx = len(lines[lineIdx])
				}
			}

		case keyDown:
			if lineIdx < len(lines)-1 {
				lineIdx++
				if colIdx > len(lines[lineIdx]) {
					colIdx = len(lines[lineIdx])
				}
			}

		case keyHome, keyCtrlHome:
			colIdx = 0

		case keyEnd, keyCtrlEnd:
			colIdx = len(lines[lineIdx])

		case keyCtrlLeft:
			if colIdx > 0 {
				colIdx--
				for colIdx > 0 && lines[lineIdx][colIdx-1] == ' ' {
					colIdx--
				}
				for colIdx > 0 && lines[lineIdx][colIdx-1] != ' ' {
					colIdx--
				}
			} else if lineIdx > 0 {
				lineIdx--
				colIdx = len(lines[lineIdx])
			}

		case keyCtrlRight:
			if colIdx < len(lines[lineIdx]) {
				for colIdx < len(lines[lineIdx]) && lines[lineIdx][colIdx] == ' ' {
					colIdx++
				}
				for colIdx < len(lines[lineIdx]) && lines[lineIdx][colIdx] != ' ' {
					colIdx++
				}
			} else if lineIdx < len(lines)-1 {
				lineIdx++
				colIdx = 0
			}

		case keyBackspace:
			if colIdx > 0 {
				lines[lineIdx] = append(lines[lineIdx][:colIdx-1], lines[lineIdx][colIdx:]...)
				colIdx--
			} else if lineIdx > 0 {
				// Merge current line into previous
				colIdx = len(lines[lineIdx-1])
				lines[lineIdx-1] = append(lines[lineIdx-1], lines[lineIdx]...)
				lines = append(lines[:lineIdx], lines[lineIdx+1:]...)
				lineIdx--
			}

		case keyDelete:
			if colIdx < len(lines[lineIdx]) {
				lines[lineIdx] = append(lines[lineIdx][:colIdx], lines[lineIdx][colIdx+1:]...)
			} else if lineIdx < len(lines)-1 {
				// Merge next line into current
				lines[lineIdx] = append(lines[lineIdx], lines[lineIdx+1]...)
				lines = append(lines[:lineIdx+1], lines[lineIdx+2:]...)
			}

		case keySpace:
			lines[lineIdx] = slices.Insert(lines[lineIdx], colIdx, ' ')
			colIdx++

		case keyRune:
			lines[lineIdx] = slices.Insert(lines[lineIdx], colIdx, ev.r)
			colIdx++
		}

		receivedInput = true

		if a.validator != nil {
			msg, ok := a.validator(joinLines())
			if !ok {
				redraw(msg)
			} else {
				redraw("")
			}
		} else {
			redraw("")
		}
		return false
	})

	if err != nil {
		return "", err
	}
	if interrupted {
		return "", ErrInterrupted
	}

	return joinLines(), nil
}
