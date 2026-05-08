package termactions

import (
	"bufio"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// EchoMode controls how typed characters are displayed during input.
type EchoMode uint8

const (
	echoNormal EchoMode = iota // characters echoed as-is
	EchoMask                   // characters echoed as * (default for Secret)
	EchoSilent                 // nothing echoed
)

// text renders an interactive single-line text prompt.
// Construct one with [Text].
type text struct {
	cfg          Config
	prefix       string
	label        string
	placeholder  string
	defaultValue string
	echo         EchoMode
	validator    func(string) (string, bool)
}

// secret renders an interactive single-line prompt for sensitive input.
// Construct one with [Secret].
type secret struct {
	text
}

// Text returns a builder for an interactive single-line text prompt.
//
//	name, err := termactions.Text().WithLabel("Project name").Render()
//	if errors.Is(err, termactions.ErrInterrupted) { ... }
func Text() *text {
	return &text{
		cfg:   pkgConfig,
		label: "Enter value",
		echo:  echoNormal,
	}
}

// Secret returns a builder for a masked prompt.
// Characters are echoed as * so the user receives visual feedback.
//
//	pass, err := termactions.Secret().WithLabel("Password").Render()
func Secret() *secret {
	return &secret{text{
		cfg:   pkgConfig,
		label: "Enter value",
		echo:  EchoMask,
	}}
}

// WithStyles overrides the [StyleMap] for this prompt.
func (t *text) WithStyles(s *StyleMap) *text {
	t.cfg.Styles = s
	return t
}

// WithPrefix overrides the default prompt prefix symbol.
func (t *text) WithPrefix(p string) *text {
	t.prefix = p
	return t
}

// WithLabel sets the prompt label shown to the user.
func (t *text) WithLabel(l string) *text {
	t.label = l
	return t
}

// WithPlaceholder sets placeholder text shown when the input is empty.
func (t *text) WithPlaceholder(p string) *text {
	t.placeholder = p
	return t
}

// WithDefaultValue sets a default value used when the user submits empty input.
func (t *text) WithDefaultValue(v string) *text {
	t.defaultValue = v
	return t
}

// WithValidator sets a validation function called on every keystroke and on submit.
// Returns a message and false to block submission, or a message and true to allow.
func (t *text) WithValidator(fn func(string) (string, bool)) *text {
	t.validator = fn
	return t
}

// WithEcho sets how typed characters are displayed.
// Defaults to [EchoMask]. Use [EchoSilent] for no visual feedback.
//
//	pass, err := termactions.Secret().WithEcho(termactions.EchoSilent).WithLabel("API Key").Render()
func (s *secret) WithEcho(m EchoMode) *secret {
	s.echo = m
	return s
}

// WithStyles overrides the [StyleMap] for this prompt.
func (s *secret) WithStyles(st *StyleMap) *secret {
	s.cfg.Styles = st
	return s
}

// WithPrefix overrides the default prompt prefix symbol.
func (s *secret) WithPrefix(p string) *secret {
	s.prefix = p
	return s
}

// WithLabel sets the prompt label shown to the user.
func (s *secret) WithLabel(l string) *secret {
	s.label = l
	return s
}

// WithValidator sets a validation function called on submit.
// Returns a message and false to block submission, or a message and true to allow.
func (s *secret) WithValidator(fn func(string) (string, bool)) *secret {
	s.validator = fn
	return s
}

// Render displays the interactive prompt and blocks until the user submits or
// cancels. Returns the entered string, or [ErrInterrupted] if Ctrl+C is pressed.
//
// In accessible mode, input is collected line-by-line.
// Validation is checked on Enter and the prompt reprints until satisfied.
func (t *text) Render() (string, error) {
	if t.cfg.Accessible {
		return t.renderAccessible()
	}
	return t.renderInteractive()
}

// renderAccessible collects input without cursor magic.
// Plain input echoes characters as typed using bufio.
// Secret echoes * per character; silent echoes nothing.
// Validation is checked on Enter and the prompt reprints on failure.
func (t *text) renderAccessible() (string, error) {
	prefix := pick(t.prefix, "(?)")
	promptLine := safeStyle(t.cfg.Styles.InputPrefix).Sprint(prefix) + " " +
		safeStyle(t.cfg.Styles.InputLabel).Sprint(t.label)

	placeholder := ""
	if t.placeholder != "" {
		placeholder = safeStyle(t.cfg.Styles.InputPlaceholder).Sprint(t.placeholder)
	}

	for {
		stdOutput.Write([]byte(promptLine + "\n"))
		if placeholder != "" {
			stdOutput.Write([]byte(placeholder + "\n"))
		}

		var result string

		if t.echo != echoNormal {
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

			type readResult struct {
				b   []byte
				err error
			}
			ch := make(chan readResult, 1)
			go func() {
				b, err := term.ReadPassword(int(os.Stdin.Fd()))
				ch <- readResult{b, err}
			}()

			select {
			case <-sigCh:
				signal.Stop(sigCh)
				return "", ErrInterrupted
			case r := <-ch:
				signal.Stop(sigCh)
				if r.err != nil {
					if isInterrupt(r.err) {
						return "", ErrInterrupted
					}
					return "", r.err
				}
				if t.echo == EchoMask {
					stdOutput.Write([]byte(strings.Repeat("*", len(r.b)) + "\n"))
				} else {
					stdOutput.Write([]byte("\n"))
				}
				result = string(r.b)
			}
		} else {
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

			select {
			case <-sigCh:
				signal.Stop(sigCh)
				return "", ErrInterrupted
			case r := <-ch:
				signal.Stop(sigCh)
				if r.err != nil {
					if isInterrupt(r.err) {
						return "", ErrInterrupted
					}
					return "", r.err
				}
				result = strings.TrimRight(r.line, "\r\n")
			}
		}

		if t.validator != nil {
			msg, ok := t.validator(result)
			if !ok {
				stdOutput.Write([]byte(safeStyle(t.cfg.Styles.InputValidationFail).Sprint(msg) + "\n\n"))
				continue
			}
		}

		if result == "" && t.defaultValue != "" {
			result = t.defaultValue
		}

		return result, nil
	}
}

// renderInteractive renders the animated single-line prompt with live redraws.
func (t *text) renderInteractive() (string, error) {
	const (
		minTermWidth  = 42
		minTermHeight = 6
	)

	var (
		cursorRow     = 0 // zero-based row of cursor within the prompt+input line
		prefix        = pick(t.prefix, "(?)")
		inBuf         []rune
		cursorPos     = 0
		interrupted   = false
		receivedInput = false
		firstRender   = true
	)

	// Guard against small terminal dimensions
	if w, h, err := termSize(); err != nil || w < minTermWidth || h < minTermHeight {
		return "", ErrTerminalTooSmall
	}

	// Build static segments
	prompt := safeStyle(t.cfg.Styles.InputPrefix).Sprint(prefix) + " " +
		safeStyle(t.cfg.Styles.InputLabel).Sprint(t.label) + ": "
	helpLine := safeStyle(t.cfg.Styles.InputHelp).Sprint("enter to confirm  •  ctrl+c to cancel")

	// displayBuf returns the string to render based on echo mode.
	displayBuf := func(buf []rune) string {
		switch t.echo {
		case EchoMask:
			return strings.Repeat("*", len(buf))
		case EchoSilent:
			return ""
		default:
			return string(buf)
		}
	}

	// buildInputContent returns the inline input content based on buffer state.
	buildInputContent := func() string {
		if len(inBuf) == 0 {
			if t.defaultValue != "" && t.placeholder != "" {
				return safeStyle(t.cfg.Styles.InputPlaceholder).Sprint(t.placeholder + " (default: " + t.defaultValue + ")")
			} else if t.defaultValue != "" {
				return safeStyle(t.cfg.Styles.InputPlaceholder).Sprint(t.defaultValue)
			} else if t.placeholder != "" {
				return safeStyle(t.cfg.Styles.InputPlaceholder).Sprint(t.placeholder)
			}
			return ""
		}
		return safeStyle(t.cfg.Styles.InputText).Sprint(displayBuf(inBuf))
	}

	redraw := func(validationMsg string) {
		termW, termH, _ := termSize()

		// Build the prompt+input line
		promptLine := prompt + buildInputContent()

		// Only show validation after user has started typing
		validationLine := ""
		if t.validator != nil && receivedInput && validationMsg != "" {
			validationLine = safeStyle(t.cfg.Styles.InputValidationFail).Sprint(validationMsg)
		}

		frameLines := []string{promptLine, "", validationLine, helpLine}
		frameHeight := totalPhysicalLines(frameLines, termW)

		// Move cursor back to row 0 of the frame
		if !firstRender {
			ansiCursorUp(cursorRow)
		}

		if termH < frameHeight || termW < minTermWidth || termH < minTermHeight {
			stdOutput.Write([]byte(
				"\r" + ansiClearScreen +
					safeStyle(t.cfg.Styles.InputValidationFail).Sprint("terminal too small to render content"),
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
		if t.echo == EchoSilent || len(inBuf) == 0 {
			stdOutput.Write([]byte("\r" + prompt))
			cursorRow = physicalLines(stripAnsi(prompt), termW) - 1
		} else {
			before := safeStyle(t.cfg.Styles.InputText).Sprint(displayBuf(inBuf[:cursorPos]))
			stdOutput.Write([]byte("\r" + prompt + before))
			plainUpToCursor := stripAnsi(prompt) + displayBuf(inBuf[:cursorPos])
			cursorRow = physicalLines(plainUpToCursor, termW) - 1
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

		case keyEnter:
			if t.validator != nil {
				msg, ok := t.validator(string(inBuf))
				if !ok {
					receivedInput = true
					redraw(msg)
					return false
				}
			}
			if len(inBuf) == 0 && t.defaultValue != "" {
				inBuf = []rune(t.defaultValue)
			}
			receivedInput = true
			return true

		case keyLeft:
			if t.echo != EchoSilent && cursorPos > 0 {
				cursorPos--
			}

		case keyRight:
			if t.echo != EchoSilent && cursorPos < len(inBuf) {
				cursorPos++
			}

		case keyHome, keyCtrlHome:
			if t.echo != EchoSilent {
				cursorPos = 0
			}

		case keyEnd, keyCtrlEnd:
			if t.echo != EchoSilent {
				cursorPos = len(inBuf)
			}

		case keyCtrlLeft:
			if t.echo == echoNormal && cursorPos > 0 {
				cursorPos--
				for cursorPos > 0 && inBuf[cursorPos-1] == ' ' {
					cursorPos--
				}
				for cursorPos > 0 && inBuf[cursorPos-1] != ' ' {
					cursorPos--
				}
			}

		case keyCtrlRight:
			if t.echo == echoNormal && cursorPos < len(inBuf) {
				for cursorPos < len(inBuf) && inBuf[cursorPos] == ' ' {
					cursorPos++
				}
				for cursorPos < len(inBuf) && inBuf[cursorPos] != ' ' {
					cursorPos++
				}
			}

		case keyBackspace:
			if t.echo == EchoSilent {
				if len(inBuf) > 0 {
					inBuf = inBuf[:len(inBuf)-1]
					cursorPos = len(inBuf)
				}
			} else if cursorPos > 0 {
				inBuf = append(inBuf[:cursorPos-1], inBuf[cursorPos:]...)
				cursorPos--
			}

		case keyDelete:
			if t.echo != EchoSilent && cursorPos < len(inBuf) {
				inBuf = append(inBuf[:cursorPos], inBuf[cursorPos+1:]...)
			}

		case keySpace:
			if t.echo != EchoSilent {
				inBuf = slices.Insert(inBuf, cursorPos, ' ')
				cursorPos++
			}

		case keyRune:
			inBuf = slices.Insert(inBuf, cursorPos, ev.r)
			cursorPos++
		}

		receivedInput = true

		if t.validator != nil {
			msg, ok := t.validator(string(inBuf))
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

	return strings.TrimRight(string(inBuf), "\r\n"), nil
}
