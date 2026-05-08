package termactions

import (
	"bufio"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// confirm renders an interactive yes/no prompt.
// Construct one with [Confirm].
type confirm struct {
	cfg        Config
	prefix     string
	label      string
	defaultVal *bool // nil = no default, user must explicitly select
}

// Confirm returns a builder for an interactive yes/no prompt.
//
//	ok, err := termactions.Confirm().WithLabel("Continue?").Render()
//	ok, err := termactions.Confirm().WithLabel("Continue?").WithDefault(true).Render()
//	if errors.Is(err, termactions.ErrInterrupted) { ... }
func Confirm() *confirm {
	return &confirm{
		cfg:   pkgConfig,
		label: "Confirm?",
	}
}

// WithStyles overrides the [StyleMap] for this prompt.
func (c *confirm) WithStyles(s *StyleMap) *confirm {
	c.cfg.Styles = s
	return c
}

// WithPrefix overrides the default prompt prefix symbol.
func (c *confirm) WithPrefix(p string) *confirm {
	c.prefix = p
	return c
}

// WithLabel sets the prompt label shown to the user.
func (c *confirm) WithLabel(l string) *confirm {
	c.label = l
	return c
}

// WithDefault pre-selects an option. If not called, the user must explicitly
// select before confirming.
func (c *confirm) WithDefault(v bool) *confirm {
	c.defaultVal = &v
	return c
}

// Render displays the interactive prompt and blocks until the user confirms or
// cancels. Returns true for yes, false for no, or [ErrInterrupted] if Ctrl+C
// is pressed.
func (c *confirm) Render() (bool, error) {
	if c.cfg.Accessible {
		return c.renderAccessible()
	}
	return c.renderInteractive()
}

// renderAccessible collects a y/n answer without ANSI cursor movement.
func (c *confirm) renderAccessible() (bool, error) {
	prefix := pick(c.prefix, "(?)")

	base := safeStyle(c.cfg.Styles.ConfirmationPrefix).Sprint(prefix) + " " +
		safeStyle(c.cfg.Styles.ConfirmationLabel).Sprint(c.label) + " "

	switch {
	case c.defaultVal == nil:
		base += safeStyle(c.cfg.Styles.ConfirmationHelp).Sprint("(type Y or N)")
	case *c.defaultVal:
		base += safeStyle(c.cfg.Styles.ConfirmationHelp).Sprint("(Y/n)")
	default:
		base += safeStyle(c.cfg.Styles.ConfirmationHelp).Sprint("(y/N)")
	}

	for {
		stdOutput.Write([]byte(base + " "))
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
			return false, ErrInterrupted
		case r := <-ch:
			signal.Stop(sigCh)
			if r.err != nil {
				if isInterrupt(r.err) {
					stdOutput.Write([]byte("\n"))
					return false, ErrInterrupted
				}
				return false, r.err
			}
			line = strings.TrimRight(r.line, "\r\n")
		}

		switch strings.ToLower(strings.TrimSpace(line)) {
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		case "":
			if c.defaultVal != nil {
				return *c.defaultVal, nil
			}
			// no default — reprompt
		}
	}
}

// renderInteractive renders the prompt label and help line.
// Y/N keys confirm directly — no need to press Enter.
// Cleans up after itself on exit.
func (c *confirm) renderInteractive() (bool, error) {
	prefix := pick(c.prefix, "(?)")
	promptLine := safeStyle(c.cfg.Styles.ConfirmationPrefix).Sprint(prefix) + " " +
		safeStyle(c.cfg.Styles.ConfirmationLabel).Sprint(c.label) + " "

	var selected *bool
	if c.defaultVal != nil {
		v := *c.defaultVal
		selected = &v
	}

	var (
		interrupted = false
		firstRender = true
		cursorRow   = 0
	)

	var helpLine string
	switch {
	case c.defaultVal == nil:
		helpLine = safeStyle(c.cfg.Styles.ConfirmationHelp).Sprint("press Y or N (selection mandatory) • ctrl+c to cancel")
	case *c.defaultVal:
		helpLine = safeStyle(c.cfg.Styles.ConfirmationHelp).Sprint("press Y or N (default: yes) • ctrl+c to cancel")
	default:
		helpLine = safeStyle(c.cfg.Styles.ConfirmationHelp).Sprint("press Y or N (default: no) • ctrl+c to cancel")
	}

	redraw := func() {
		termW, _, _ := termSize()

		frameLines := []string{promptLine, helpLine}
		frameHeight := totalPhysicalLines(frameLines, termW)

		// Move cursor back to row 0 of the frame
		if !firstRender {
			ansiCursorUp(cursorRow)
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

		// Position cursor at end of prompt line by reprinting it
		stdOutput.Write([]byte("\r" + promptLine))
		cursorRow = physicalLines(stripAnsi(promptLine), termW) - 1

		stdOutput.Write([]byte(ansiShowCursor))
		firstRender = false
	}

	// Hide cursor, defer cleanup
	stdOutput.Write([]byte("\r" + ansiHideCursor))
	defer func() {
		ansiCursorUp(cursorRow)
		stdOutput.Write([]byte("\r" + ansiClearScreen + ansiReset + ansiShowCursor))
	}()

	// Initial render
	redraw()

	// Intercept keyboard events & handle them
	err := listenKeys(func(ev keyEvent) (stop bool) {
		switch ev.code {
		case keyCtrlC:
			interrupted = true
			return true

		case keyEnter:
			if selected == nil {
				return false // block until user presses Y or N
			}
			return true

		case keyRune:
			switch ev.r {
			case 'y', 'Y':
				v := true
				selected = &v
				return true
			case 'n', 'N':
				v := false
				selected = &v
				return true
			}
		}

		redraw()
		return false
	})

	if err != nil {
		return false, err
	}
	if interrupted {
		return false, ErrInterrupted
	}

	if selected == nil {
		return false, nil
	}
	return *selected, nil
}
