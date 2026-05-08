package termactions

import (
	"bufio"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

// multiSelect renders an interactive multi-selection prompt.
// Construct one with [MultiSelect].
type multiSelect struct {
	cfg             Config
	prefix          string
	label           string
	choices         []Choice
	preSelected     []string
	cursorIndicator string
	selectionMarker string
	pageSize        int
	selectedChoices []Choice
	validator       func([]Choice) (string, bool)
}

// MultiSelect returns a builder for an interactive multi-selection prompt.
//
//	choices, err := termactions.MultiSelect().WithLabel("Pick some").WithChoices(choices).Render()
//	if errors.Is(err, termactions.ErrInterrupted) { ... }
func MultiSelect() *multiSelect {
	return &multiSelect{
		cfg:             pkgConfig,
		label:           "Select options",
		choices:         []Choice{},
		cursorIndicator: ">",
		selectionMarker: "*",
		pageSize:        10,
	}
}

// WithStyles overrides the [StyleMap] for this prompt.
func (s *multiSelect) WithStyles(sm *StyleMap) *multiSelect {
	s.cfg.Styles = sm
	return s
}

// WithPrefix overrides the default prompt prefix symbol.
func (s *multiSelect) WithPrefix(p string) *multiSelect {
	s.prefix = p
	return s
}

// WithLabel sets the prompt label shown to the user.
func (s *multiSelect) WithLabel(l string) *multiSelect {
	s.label = l
	return s
}

// WithChoices sets the list of choices available for selection.
func (s *multiSelect) WithChoices(ch []Choice) *multiSelect {
	s.choices = ch
	return s
}

// WithDefaultChoices sets the list of choices to be selected by default.
func (m *multiSelect) WithSelectedChoices(values []string) *multiSelect {
	m.preSelected = values
	return m
}

// WithPageSize sets the number of choices visible at once.
func (s *multiSelect) WithPageSize(n int) *multiSelect {
	s.pageSize = n
	return s
}

// WithCursorIndicator overrides the cursor indicator symbol.
func (s *multiSelect) WithCursorIndicator(ind string) *multiSelect {
	s.cursorIndicator = ind
	return s
}

// WithSelectionMarker overrides the selection marker symbol.
func (s *multiSelect) WithSelectionMarker(mrk string) *multiSelect {
	s.selectionMarker = mrk
	return s
}

// WithValidator sets a validator called on enter.
// Use [ValidateMultiSelectRequired] or a custom func([]Choice) (string, bool).
func (s *multiSelect) WithValidator(v func([]Choice) (string, bool)) *multiSelect {
	s.validator = v
	return s
}

// Render displays the prompt and blocks until the user confirms or cancels.
// Returns the selected choices, or [ErrInterrupted] if Ctrl+C is pressed.
//
// In accessible mode, choices are printed as a numbered list and the user
// types comma-separated indices. In interactive mode, choices are navigated
// with arrow keys and toggled with space.
func (s *multiSelect) Render() ([]Choice, error) {
	if len(s.choices) == 0 {
		return nil, ErrNoSelectionChoices
	}

	// Pre-populate selected choices from WithSelectedChoices
	preSelectedSet := make(map[string]bool)
	for _, v := range s.preSelected {
		preSelectedSet[v] = true
	}
	for _, c := range s.choices {
		if preSelectedSet[c.Value] {
			s.selectedChoices = append(s.selectedChoices, c)
		}
	}

	if s.cfg.Accessible {
		return s.renderAccessible()
	}
	return s.renderInteractive()
}

// isSelected reports whether c is in the current selection.
func (s *multiSelect) isSelected(c Choice) bool {
	for _, sel := range s.selectedChoices {
		if sel.Value == c.Value {
			return true
		}
	}
	return false
}

// toggleChoice adds c to the selection if not present, or removes it if present.
func (s *multiSelect) toggleChoice(c Choice) {
	for i, sel := range s.selectedChoices {
		if sel.Value == c.Value {
			s.selectedChoices = append(s.selectedChoices[:i], s.selectedChoices[i+1:]...)
			return
		}
	}
	s.selectedChoices = append(s.selectedChoices, c)
}

// renderAccessible prints a numbered list and collects the user's choices by
// comma-separated indices. It uses a 1-based index printed next to each label.
func (s *multiSelect) renderAccessible() ([]Choice, error) {

	// Print the header
	prefix := pick(s.prefix, "(?)")
	stdOutput.Write([]byte(
		safeStyle(s.cfg.Styles.SelectionPrefix).Sprint(prefix+" ") +
			safeStyle(s.cfg.Styles.SelectionLabel).Sprint(s.label) + "\n",
	))

	// Print numbered choices
	width := len(strconv.Itoa(len(s.choices)))
	for i, c := range s.choices {
		num := safeStyle(s.cfg.Styles.SelectionSearchHint).Sprintf("%*d. ", width, i+1)
		label := safeStyle(s.cfg.Styles.SelectionItemNormalLabel).Sprint(c.Label)
		marker := ""
		for _, sel := range s.selectedChoices {
			if sel.Value == c.Value {
				marker = safeStyle(s.cfg.Styles.SelectionItemSelectedMarker).Sprint(" *")
				break
			}
		}
		stdOutput.Write([]byte("  " + num + label + marker + "\n"))
	}

	promptStr := safeStyle(s.cfg.Styles.SelectionPrefix).Sprint(prefix) + " " +
		safeStyle(s.cfg.Styles.SelectionLabel).Sprintf("Enter numbers separated by commas: ")

	for {
		stdOutput.Write([]byte(promptStr))

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

		type readResult struct {
			line string
			err  error
		}
		ch := make(chan readResult, 1)
		go func() {
			line, err := bufio.NewReader(os.Stdin).ReadString('\n')
			ch <- readResult{line, err}
		}()

		var line string
		select {
		case <-sigCh:
			signal.Stop(sigCh)
			stdOutput.Write([]byte("\n"))
			return nil, ErrInterrupted
		case r := <-ch:
			signal.Stop(sigCh)
			if r.err != nil {
				if isInterrupt(r.err) {
					stdOutput.Write([]byte("\n"))
					return nil, ErrInterrupted
				}
				return nil, r.err
			}
			line = strings.TrimSpace(strings.TrimRight(r.line, "\r\n"))
		}

		if line == "" {
			if len(s.selectedChoices) > 0 {
				if s.validator != nil {
					if msg, ok := s.validator(s.selectedChoices); !ok {
						stdOutput.Write([]byte(safeStyle(s.cfg.Styles.SelectionValidationFail).Sprint(msg) + "\n"))
						continue
					}
				}
				return s.selectedChoices, nil
			}
			stdOutput.Write([]byte(safeStyle(s.cfg.Styles.SelectionValidationFail).Sprint("please enter at least one number") + "\n"))
			continue
		}

		// Parse comma-separated indices
		var chosen []Choice
		valid := true
		for _, part := range strings.Split(line, ",") {
			part = strings.TrimSpace(part)
			n, err := strconv.Atoi(part)
			if err != nil || n < 1 || n > len(s.choices) {
				stdOutput.Write([]byte(
					safeStyle(s.cfg.Styles.SelectionValidationFail).
						Sprintf("invalid choice %q — enter numbers between 1 and %d\n", part, len(s.choices)),
				))
				valid = false
				break
			}
			chosen = append(chosen, s.choices[n-1])
		}
		if !valid {
			continue
		}

		if s.validator != nil {
			if msg, ok := s.validator(chosen); !ok {
				stdOutput.Write([]byte(safeStyle(s.cfg.Styles.SelectionValidationFail).Sprint(msg) + "\n"))
				continue
			}
		}

		return chosen, nil
	}
}

// renderInteractive renders a navigable list with search. Arrow keys and
// vi-keys move the cursor, space toggles selection, enter confirms.
func (s *multiSelect) renderInteractive() ([]Choice, error) {
	const (
		minTermWidth  = 42
		minTermHeight = 12
	)
	var (
		interrupted     = false
		searchQuery     = ""
		searchMode      = false
		filteredChoices = s.choices
		nav             = &selectionNav{}
		valMessage      = ""
		prevHeight      = 0
	)

	// Initialize navigation
	nav.reset(len(filteredChoices), min(s.pageSize, len(filteredChoices)))

	// Guard against small terminal dimensions
	if w, h, err := termSize(); err != nil || w < minTermWidth || h < minTermHeight {
		return nil, ErrTerminalTooSmall
	}

	// Build the header lines
	promptLine := safeStyle(s.cfg.Styles.SelectionPrefix).Sprint(pick(s.prefix, "(?)")) + " " +
		safeStyle(s.cfg.Styles.SelectionLabel).Sprint(s.label)
	searchLabel := safeStyle(s.cfg.Styles.SelectionSearchLabel).Sprint("Search: ")
	headerLines := []string{promptLine, ""}

	// Multi-Select Prompt Renderer
	redraw := func() {
		newW, newH, _ := termSize()

		// Build the current search line
		searchLine := searchLabel + safeStyle(s.cfg.Styles.SelectionSearchText).Sprint(searchQuery)
		if searchMode {
			searchLine += safeStyle(s.cfg.Styles.SelectionSearchHint).Sprint(" • " + strconv.Itoa(len(filteredChoices)) + " hits")
		}
		searchLine += safeStyle(s.cfg.Styles.SelectionSearchHint).Sprint(" (" + strconv.Itoa(len(s.selectedChoices)) + " selected)")

		// Update the header lines & compute the frame height for header
		headerLines[1] = searchLine
		headerLinesHeight := totalPhysicalLines(headerLines, newW)

		// Build the footer lines & compute the frame height for footer
		footerLines := []string{""}
		footerLines = append(footerLines, safeStyle(s.cfg.Styles.SelectionValidationFail).Sprint(valMessage))
		if searchMode {
			footerLines = append(footerLines, safeStyle(s.cfg.Styles.SelectionHelp).Sprint("↑/↓ move • space toggle • enter confirm"))
			footerLines = append(footerLines, safeStyle(s.cfg.Styles.SelectionHelp).Sprint("type to search (esc/tab nav)"))
		} else {
			footerLines = append(footerLines, safeStyle(s.cfg.Styles.SelectionHelp).Sprint("↑/↓ move • space toggle • enter confirm"))
			footerLines = append(footerLines, safeStyle(s.cfg.Styles.SelectionHelp).Sprint("tab to search"))
		}
		footerLinesHeight := totalPhysicalLines(footerLines, newW)

		// Compute page size & reset navigation if needed
		pageSize := min(s.pageSize, len(filteredChoices), newH-headerLinesHeight-footerLinesHeight)
		if pageSize != nav.pageSize && pageSize > 0 {
			nav.reset(len(filteredChoices), pageSize)
		}

		// Build contentLines
		var contentLines []string
		contentLines = append(contentLines, headerLines...)

		// Build content for the visible choices list & pad the rest with empty lines
		for i := nav.startIdx; i < nav.endIdx; i++ {
			contentLines = append(contentLines, renderSelectionChoice(
				filteredChoices[i],
				i == nav.cursorIdx,
				s.isSelected(filteredChoices[i]),
				newW-1,
				s.cursorIndicator,
				s.selectionMarker,
				s.cfg.Styles),
			)
		}

		// Pad the rest to maintain consistent height
		for i := nav.endIdx - nav.startIdx; i < nav.pageSize; i++ {
			contentLines = append(contentLines, "")
		}
		contentLines = append(contentLines, footerLines...)

		// Compute new frame's physical height at current width
		newHeight := totalPhysicalLines(contentLines, newW)

		if newH < newHeight || newW < minTermWidth || newH < minTermHeight {
			ansiCursorUp(prevHeight)
			stdOutput.Write([]byte(
				"\r" + ansiClearScreen +
					safeStyle(s.cfg.Styles.SelectionItemCurrentMarker).Sprint("terminal too small to render content"),
			))
			return
		}

		// Move up by the previous frame's physical height to overwrite it
		if prevHeight > 0 {
			ansiCursorUp(prevHeight)
		}

		// Write new frame, clearing every physical row including wrapped continuations
		var b strings.Builder
		for i, line := range contentLines {
			if i == len(contentLines)-1 {
				b.WriteString("\r" + line + ansiClearLine)
			} else {
				b.WriteString("\r" + line + ansiClearLine + "\n")
			}
		}
		b.WriteString(ansiClearScreen)

		stdOutput.Write([]byte(b.String()))
		prevHeight = newHeight - 1
	}

	// Prep for render, hide cursor, defer cleanup
	stdOutput.Write([]byte("\r" + ansiHideCursor))
	defer func() {
		ansiCursorUp(prevHeight)
		stdOutput.Write([]byte("\r" + ansiClearScreen + ansiReset + ansiShowCursor))
	}()

	// Initial render
	redraw()

	// Handle user input & redraw per keystroke
	err := listenKeys(func(ev keyEvent) (stop bool) {
		switch ev.code {
		case keyCtrlC:
			interrupted = true
			return true
		case keyUp:
			nav.up(len(filteredChoices))
		case keyDown:
			nav.down(len(filteredChoices))
		case keyTab:
			searchMode = !searchMode
		case keyEscape:
			searchMode = false
		case keyEnter:
			if s.validator != nil {
				if msg, ok := s.validator(s.selectedChoices); !ok {
					valMessage = msg
					break
				}
			}
			return true
		case keySpace:
			if len(filteredChoices) == 0 {
				valMessage = "no choices available"
				break
			}
			s.toggleChoice(filteredChoices[nav.cursorIdx])
			valMessage = ""
		case keyBackspace:
			if searchMode && len(searchQuery) > 0 {
				searchQuery = searchQuery[:len(searchQuery)-1]
				filteredChoices = filterSelectionChoices(s.choices, searchQuery)
				nav.reset(len(filteredChoices), nav.pageSize)
			}
		case keyRune:
			if searchMode {
				searchQuery += string(ev.r)
				filteredChoices = filterSelectionChoices(s.choices, searchQuery)
				nav.reset(len(filteredChoices), nav.pageSize)
			} else {
				switch ev.r {
				case 'j', 'l':
					nav.down(len(filteredChoices))
				case 'k', 'h':
					nav.up(len(filteredChoices))
				}
			}
		}
		redraw()
		return false
	})

	// Handle errors, edge cases, interrupts and return selected choices
	if err != nil {
		return nil, err
	}
	if interrupted {
		return nil, ErrInterrupted
	}
	return s.selectedChoices, nil
}
