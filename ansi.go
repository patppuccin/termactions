package termactions

import (
	"strconv"
)

const (
	ansiHideCursor = "\033[?25l"
	ansiShowCursor = "\033[?25h"

	ansiReset       = "\033[0m\033[0 q"
	ansiClearLine   = "\033[K"
	ansiClearScreen = "\033[J"
)

// ansiCursorUp moves the cursor n positions up.
func ansiCursorUp(n int) {
	if n > 0 {
		stdOutput.Write([]byte("\033[" + strconv.Itoa(n) + "A"))
	}
}
