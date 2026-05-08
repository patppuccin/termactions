package termactions

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

// singleSelect renders an interactive single-selection prompt.
// Construct one with [Select].
type singleSelect struct {
	cfg             Config
	prefix          string
	label           string
	choices         []Choice
	preSelected     *string
	cursorIndicator string
	selectionMarker string
	pageSize        int
	selectedChoice  Choice
	validator       func(Choice) (string, bool)
}

// Select returns a builder for an interactive single-selection prompt.
//
//	choice, err := termactions.Select().WithLabel("Pick one").WithChoices(choices).Render()
//	if errors.Is(err, termactions.ErrInterrupted) { ... }
func Select() *singleSelect {
	return &singleSelect{
		cfg:             pkgConfig,
		label:           "Select an option",
		choices:         []Choice{},
		cursorIndicator: ">",
		selectionMarker: "*",
		pageSize:        10,
	}
}

// WithStyles overrides the [StyleMap] for this prompt.
func (s *singleSelect) WithStyles(sm *StyleMap) *singleSelect {
	s.cfg.Styles = sm
	return s
}

// WithPrefix overrides the default prompt prefix symbol.
func (s *singleSelect) WithPrefix(p string) *singleSelect {
	s.prefix = p
	return s
}

// WithLabel sets the prompt label shown to the user.
func (s *singleSelect) WithLabel(l string) *singleSelect {
	s.label = l
	return s
}

// WithChoices sets the list of choices available for selection.
func (s *singleSelect) WithChoices(ch []Choice) *singleSelect {
	s.choices = ch
	return s
}

// WithSelectedChoice pre-selects a choice by its value.
func (s *singleSelect) WithSelectedChoice(value string) *singleSelect {
	s.preSelected = &value
	return s
}

// WithPageSize sets the number of choices visible at once.
func (s *singleSelect) WithPageSize(n int) *singleSelect {
	s.pageSize = n
	return s
}

// WithCursorIndicator overrides the cursor indicator symbol.
func (s *singleSelect) WithCursorIndicator(ind string) *singleSelect {
	s.cursorIndicator = ind
	return s
}

// WithSelectionMarker overrides the selection marker symbol.
func (s *singleSelect) WithSelectionMarker(mrk string) *singleSelect {
	s.selectionMarker = mrk
	return s
}

// WithValidator sets a validator called on enter.
// Use [ValidateSelectRequired] or a custom func(Choice) (string, bool).
func (s *singleSelect) WithValidator(v func(Choice) (string, bool)) *singleSelect {
	s.validator = v
	return s
}

// Render displays the prompt and blocks until the user confirms or cancels.
// Returns the selected [Choice], or [ErrInterrupted] if Ctrl+C is pressed.
//
// In accessible mode, choices are printed as a numbered list and the user
// types the index. In interactive mode, choices are navigated with arrow keys.
func (s *singleSelect) Render() (Choice, error) {
	if len(s.choices) == 0 {
		return Choice{}, ErrNoSelectionChoices
	}
	if s.cfg.Accessible {
		return s.renderAccessible()
	}
	return s.renderInteractive()
}

// renderAccessible prints a numbered list and collects the user's choice by index.
// It uses a 1-based index, printed next to the choice label.
func (s *singleSelect) renderAccessible() (Choice, error) {

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
		stdOutput.Write([]byte("  " + num + label + "\n"))
	}

	// Build the prompt
	hint := ""
	if s.preSelected != nil {
		for i, c := range s.choices {
			if c.Value == *s.preSelected {
				hint = fmt.Sprintf(" (default %d) ", i+1)
				break
			}
		}
	}
	promptStr := safeStyle(s.cfg.Styles.SelectionPrefix).Sprint(prefix) + " " +
		safeStyle(s.cfg.Styles.SelectionLabel).Sprintf("Choose between 1 and %d%s: ", len(s.choices), hint)

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
			return Choice{}, ErrInterrupted
		case r := <-ch:
			signal.Stop(sigCh)
			if r.err != nil {
				if isInterrupt(r.err) {
					stdOutput.Write([]byte("\n"))
					return Choice{}, ErrInterrupted
				}
				return Choice{}, r.err
			}
			line = strings.TrimSpace(strings.TrimRight(r.line, "\r\n"))
		}

		// Empty input uses default if set
		if line == "" {
			if s.preSelected != nil {
				for _, c := range s.choices {
					if c.Value == *s.preSelected {
						if s.validator != nil {
							if msg, ok := s.validator(c); !ok {
								stdOutput.Write([]byte(safeStyle(s.cfg.Styles.SelectionValidationFail).Sprint(msg) + "\n"))
								continue
							}
						}
						return c, nil
					}
				}
			}
			stdOutput.Write([]byte(safeStyle(s.cfg.Styles.SelectionValidationFail).Sprint("please enter a number") + "\n"))
			continue
		}

		// Parse number
		n, err := strconv.Atoi(line)
		if err != nil || n < 1 || n > len(s.choices) {
			stdOutput.Write([]byte(
				safeStyle(s.cfg.Styles.SelectionValidationFail).
					Sprintf("enter a number between 1 and %d\n", len(s.choices)),
			))
			continue
		}

		chosen := s.choices[n-1]

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
// vi-keys move the cursor, space selects, enter confirms.
func (s *singleSelect) renderInteractive() (Choice, error) {
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
		return Choice{}, ErrTerminalTooSmall
	}

	// Build the header lines
	promptLine := safeStyle(s.cfg.Styles.SelectionPrefix).Sprint(pick(s.prefix, "(?)")) + " " +
		safeStyle(s.cfg.Styles.SelectionLabel).Sprint(s.label)
	searchLabel := safeStyle(s.cfg.Styles.SelectionSearchLabel).Sprint("Search: ")
	headerLines := []string{promptLine, ""}

	// Selection Prompt Renderer
	redraw := func() {
		newW, newH, _ := termSize()

		// Build the current search line
		searchLine := searchLabel + safeStyle(s.cfg.Styles.SelectionSearchText).Sprint(searchQuery)
		if searchMode {
			searchLine += safeStyle(s.cfg.Styles.SelectionSearchHint).Sprint(" • " + strconv.Itoa(len(filteredChoices)) + " hits")
		}
		if s.selectedChoice != (Choice{}) {
			searchLine += safeStyle(s.cfg.Styles.SelectionSearchHint).Sprint(" (1 selected)")
		} else {
			searchLine += safeStyle(s.cfg.Styles.SelectionSearchHint).Sprint(" (0 selected)")
		}

		// Update the header lines & compute the frame height for header
		headerLines[1] = searchLine
		headerLinesHeight := totalPhysicalLines(headerLines, newW)

		// Build the footer lines & compute the frame height for footer
		footerLines := []string{""}
		footerLines = append(footerLines, safeStyle(s.cfg.Styles.SelectionValidationFail).Sprint(valMessage))
		if searchMode {
			footerLines = append(footerLines, safeStyle(s.cfg.Styles.SelectionHelp).Sprint("↑/↓ move • space select • enter confirm"))
			footerLines = append(footerLines, safeStyle(s.cfg.Styles.SelectionHelp).Sprint("type to search (esc/tab nav)"))
		} else {
			footerLines = append(footerLines, safeStyle(s.cfg.Styles.SelectionHelp).Sprint("↑/↓ move • space select • enter confirm"))
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
				filteredChoices[i].Value == s.selectedChoice.Value,
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

	// Apply default selection by value
	if s.preSelected != nil {
		for _, c := range s.choices {
			if c.Value == *s.preSelected {
				s.selectedChoice = c
				break
			}
		}
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
				if msg, ok := s.validator(s.selectedChoice); !ok {
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
			cur := filteredChoices[nav.cursorIdx]
			if s.selectedChoice.Value == cur.Value {
				s.selectedChoice = Choice{}
			} else {
				s.selectedChoice = cur
			}
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

	// Handle errors, edge cases, interrupts and return selected choice
	if err != nil {
		return Choice{}, err
	}
	if interrupted {
		return Choice{}, ErrInterrupted
	}
	return s.selectedChoice, nil
}
