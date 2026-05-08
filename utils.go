package termactions

import (
	"errors"
	"io"
	"os"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

// termSize returns the current terminal width and height in columns and rows.
func termSize() (int, int, error) {
	return term.GetSize(int(os.Stdout.Fd()))
}

// safeStyle returns s if non-nil, otherwise a no-op Reset style.
// Guards against nil fields on a partially constructed [StyleMap].
func safeStyle(s *color.Color) *color.Color {
	if s != nil {
		return s
	}
	return color.New(color.Reset)
}

// pick returns val if non-empty, otherwise fallback.
func pick(val, fallback string) string {
	if val != "" {
		return val
	}
	return fallback
}

// isInterrupt reports whether err represents a user cancellation.
// Covers io.EOF from bufio, syscall.EINTR from term on Unix,
// and interrupted system call errors on Windows.
func isInterrupt(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, io.EOF) ||
		errors.Is(err, syscall.EINTR) ||
		strings.Contains(err.Error(), "interrupted")
}

// stripAnsi removes ANSI escape sequences from s, returning plain text.
// Handles both CSI sequences (\033[...m) and non-CSI escapes (\0337, \0338).
func stripAnsi(s string) string {
	var out strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\033' && i+1 < len(s) {
			if s[i+1] == '[' {
				// CSI sequence: \033[ ... letter
				i += 2
				for i < len(s) && !((s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z')) {
					i++
				}
				i++
			} else {
				// Non-CSI escape: \033 + single char (e.g. \0337, \0338)
				i += 2
			}
		} else {
			out.WriteByte(s[i])
			i++
		}
	}
	return out.String()
}

// physicalLines returns the number of terminal rows s occupies at termWidth,
// after stripping ANSI escape sequences from s.
func physicalLines(s string, termWidth int) int {
	visible := runewidth.StringWidth(stripAnsi(s))
	if visible == 0 {
		return 1
	}
	return (visible + termWidth - 1) / termWidth
}

// totalPhysicalLines returns the total terminal rows occupied by all lines
// at termWidth, accounting for ANSI escape sequences and line wrapping.
func totalPhysicalLines(lines []string, termWidth int) int {
	total := 0
	for _, l := range lines {
		total += physicalLines(l, termWidth)
	}
	return total
}

// TruncToWidth truncates content to fit within availableWidth columns,
// appending an ellipsis (…) if truncation occurs. Rune-aware and
// handles multi-byte and wide characters correctly.
func TruncToWidth(content string, availableWidth int) string {
	if availableWidth <= 1 {
		return "…"
	}
	if runewidth.StringWidth(content) <= availableWidth {
		return content
	}
	var truncated strings.Builder
	used := 0
	for _, r := range content {
		rw := runewidth.RuneWidth(r)
		if used+rw > availableWidth-1 {
			break
		}
		truncated.WriteString(string(r))
		used += rw
	}
	return truncated.String() + "…"
}
