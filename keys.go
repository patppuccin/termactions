package termactions

import (
	"bufio"
	"os"
	"time"

	"golang.org/x/term"
)

// keyCode represents a parsed terminal key event.
type keyCode int

const (
	keyRune      keyCode = iota // printable character
	keyTab                      // \x09
	keySpace                    // \x20
	keyEnter                    // \r or \n
	keyBackspace                // \x7f or \x08
	keyDelete                   // \x1b[3~
	keyLeft                     // \x1b[D or \x1bOD
	keyRight                    // \x1b[C or \x1bOC
	keyUp                       // \x1b[A or \x1bOA
	keyDown                     // \x1b[B or \x1bOB
	keyHome                     // \x1b[H, \x1b[1~, or \x1bOH
	keyEnd                      // \x1b[F, \x1b[4~, or \x1bOF
	keyEscape                   // standalone \x1b (distinguished via timeout)
	keyCtrlC                    // \x03
	keyCtrlD                    // \x04
	keyCtrlLeft                 // \x1b[1;5D
	keyCtrlRight                // \x1b[1;5C
	keyCtrlHome                 // \x1b[1;5H
	keyCtrlEnd                  // \x1b[1;5F
	keyUnknown
)

// keyEvent is a parsed key press.
type keyEvent struct {
	code keyCode
	r    rune // set when code == keyRune
}

// escTimeout is how long to wait after a bare \x1b before treating it as
// a standalone Escape keypress rather than the start of an escape sequence.
// 50ms is the widely-used standard (bubbletea, readline, tcell all use this).
const escTimeout = 50 * time.Millisecond

// keyReader wraps stdin in raw mode and provides single-keypress reads.
// Call close() when done to restore the terminal.
type keyReader struct {
	fd       int
	oldState *term.State
	r        *bufio.Reader
}

// newKeyReader puts stdin into raw mode and returns a keyReader.
func newKeyReader() (*keyReader, error) {
	fd := int(os.Stdin.Fd())
	old, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	return &keyReader{
		fd:       fd,
		oldState: old,
		r:        bufio.NewReaderSize(os.Stdin, 64),
	}, nil
}

// close restores the terminal to its original state.
func (kr *keyReader) close() {
	term.Restore(kr.fd, kr.oldState) //nolint:errcheck
}

// read blocks until a key is pressed and returns a keyEvent.
// It handles Escape ambiguity by attempting a short buffered read after
// a bare \x1b — if no further bytes arrive within escTimeout, it returns
// keyEscape; otherwise it reads the full sequence and parses it.
func (kr *keyReader) read() (keyEvent, error) {
	first, err := kr.r.ReadByte()
	if err != nil {
		return keyEvent{code: keyUnknown}, err
	}

	// Not an escape byte — handle immediately.
	if first != 0x1b {
		return parseSingleOrUTF8(first, kr.r)
	}

	// It's 0x1b. Peek ahead with a timeout to decide: sequence or bare Escape.
	type peekResult struct {
		b   byte
		err error
	}
	ch := make(chan peekResult, 1)
	go func() {
		b, err := kr.r.ReadByte()
		ch <- peekResult{b, err}
	}()

	select {
	case <-time.After(escTimeout):
		// Nothing followed within the timeout — standalone Escape.
		return keyEvent{code: keyEscape}, nil

	case peek := <-ch:
		if peek.err != nil {
			return keyEvent{code: keyEscape}, nil
		}

		switch peek.b {
		case 'O':
			// SS3 sequences: \x1bO... — xterm application cursor mode, tmux, VT100.
			third, err := kr.r.ReadByte()
			if err != nil {
				return keyEvent{code: keyEscape}, nil
			}
			switch third {
			case 'A':
				return keyEvent{code: keyUp}, nil
			case 'B':
				return keyEvent{code: keyDown}, nil
			case 'C':
				return keyEvent{code: keyRight}, nil
			case 'D':
				return keyEvent{code: keyLeft}, nil
			case 'H':
				return keyEvent{code: keyHome}, nil
			case 'F':
				return keyEvent{code: keyEnd}, nil
			}
			return keyEvent{code: keyUnknown}, nil

		case '[':
			// CSI sequences: \x1b[...
			return kr.readCSI()

		default:
			// Unrecognised sequence after \x1b (e.g. Alt+key — not used by termactions yet).
			return keyEvent{code: keyUnknown}, nil
		}
	}
}

// readCSI reads the remainder of a CSI sequence (\x1b[ already consumed)
// and maps it to a keyEvent.
func (kr *keyReader) readCSI() (keyEvent, error) {
	// Read up to 6 bytes — enough for any sequence termactions handles.
	// CSI sequences terminate on a final byte in range 0x40–0x7E.
	buf := make([]byte, 0, 6)
	for len(buf) < 6 {
		b, err := kr.r.ReadByte()
		if err != nil {
			return keyEvent{code: keyUnknown}, err
		}
		buf = append(buf, b)
		if b >= 0x40 && b <= 0x7e {
			break
		}
	}

	switch {
	case len(buf) == 1 && buf[0] == 'A':
		return keyEvent{code: keyUp}, nil
	case len(buf) == 1 && buf[0] == 'B':
		return keyEvent{code: keyDown}, nil
	case len(buf) == 1 && buf[0] == 'C':
		return keyEvent{code: keyRight}, nil
	case len(buf) == 1 && buf[0] == 'D':
		return keyEvent{code: keyLeft}, nil
	case len(buf) == 1 && buf[0] == 'H':
		return keyEvent{code: keyHome}, nil
	case len(buf) == 1 && buf[0] == 'F':
		return keyEvent{code: keyEnd}, nil

	// Home: \x1b[1~
	case len(buf) == 2 && buf[0] == '1' && buf[1] == '~':
		return keyEvent{code: keyHome}, nil
	// End: \x1b[4~
	case len(buf) == 2 && buf[0] == '4' && buf[1] == '~':
		return keyEvent{code: keyEnd}, nil
	// Delete: \x1b[3~
	case len(buf) == 2 && buf[0] == '3' && buf[1] == '~':
		return keyEvent{code: keyDelete}, nil

	// Ctrl+Left: \x1b[1;5D
	case len(buf) == 4 && buf[0] == '1' && buf[1] == ';' && buf[2] == '5' && buf[3] == 'D':
		return keyEvent{code: keyCtrlLeft}, nil
	// Ctrl+Right: \x1b[1;5C
	case len(buf) == 4 && buf[0] == '1' && buf[1] == ';' && buf[2] == '5' && buf[3] == 'C':
		return keyEvent{code: keyCtrlRight}, nil
	// Ctrl+Home: \x1b[1;5H
	case len(buf) == 4 && buf[0] == '1' && buf[1] == ';' && buf[2] == '5' && buf[3] == 'H':
		return keyEvent{code: keyCtrlHome}, nil
	// Ctrl+End: \x1b[1;5F
	case len(buf) == 4 && buf[0] == '1' && buf[1] == ';' && buf[2] == '5' && buf[3] == 'F':
		return keyEvent{code: keyCtrlEnd}, nil
	}

	return keyEvent{code: keyUnknown}, nil
}

// parseSingleOrUTF8 handles a non-escape first byte: control/ASCII or the
// start of a multi-byte UTF-8 rune. Continuation bytes are read from r.
func parseSingleOrUTF8(first byte, r *bufio.Reader) (keyEvent, error) {
	switch first {
	case 0x03:
		return keyEvent{code: keyCtrlC}, nil
	case 0x04:
		return keyEvent{code: keyCtrlD}, nil
	case 0x0d, 0x0a:
		return keyEvent{code: keyEnter}, nil
	case 0x7f, 0x08:
		return keyEvent{code: keyBackspace}, nil
	case 0x09:
		return keyEvent{code: keyTab}, nil
	case 0x20:
		return keyEvent{code: keySpace}, nil
	}

	// Printable ASCII.
	if first < 0x80 {
		if first >= 0x20 {
			return keyEvent{code: keyRune, r: rune(first)}, nil
		}
		return keyEvent{code: keyUnknown}, nil
	}

	// Multi-byte UTF-8: determine expected sequence length from the leading byte.
	var seqLen int
	switch {
	case first&0xE0 == 0xC0:
		seqLen = 2
	case first&0xF0 == 0xE0:
		seqLen = 3
	case first&0xF8 == 0xF0:
		seqLen = 4
	default:
		return keyEvent{code: keyUnknown}, nil
	}

	buf := make([]byte, seqLen)
	buf[0] = first
	for i := 1; i < seqLen; i++ {
		b, err := r.ReadByte()
		if err != nil {
			return keyEvent{code: keyUnknown}, err
		}
		buf[i] = b
	}

	rv, _ := decodeRune(buf)
	if rv == 0xFFFD {
		return keyEvent{code: keyUnknown}, nil
	}
	return keyEvent{code: keyRune, r: rv}, nil
}

// decodeRune decodes the first UTF-8 rune in b.
// Returns unicode.ReplacementChar (0xFFFD) on invalid input.
func decodeRune(b []byte) (rune, int) {
	if len(b) == 0 {
		return 0xFFFD, 0
	}
	c := b[0]
	switch {
	case c < 0x80:
		return rune(c), 1
	case c < 0xC0:
		return 0xFFFD, 1
	case c < 0xE0:
		if len(b) < 2 {
			return 0xFFFD, 1
		}
		return rune(c&0x1F)<<6 | rune(b[1]&0x3F), 2
	case c < 0xF0:
		if len(b) < 3 {
			return 0xFFFD, 1
		}
		return rune(c&0x0F)<<12 | rune(b[1]&0x3F)<<6 | rune(b[2]&0x3F), 3
	default:
		if len(b) < 4 {
			return 0xFFFD, 1
		}
		return rune(c&0x07)<<18 | rune(b[1]&0x3F)<<12 | rune(b[2]&0x3F)<<6 | rune(b[3]&0x3F), 4
	}
}

// listenKeys calls fn for each key press until fn returns true (stop) or an error.
// Puts stdin into raw mode for the duration of the call.
func listenKeys(fn func(keyEvent) (stop bool)) error {
	kr, err := newKeyReader()
	if err != nil {
		return err
	}
	defer kr.close()

	for {
		ev, err := kr.read()
		if err != nil {
			return err
		}
		if fn(ev) {
			return nil
		}
	}
}
