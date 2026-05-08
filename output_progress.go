package termactions

import (
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/mattn/go-runewidth"
)

// Progress bar pattern presets.
var (
	ProgressDefault = ProgressPattern{DoneChar: "╍", PendingChar: "╌", PadLeft: "[", PadRight: "]"}
	ProgressBlock   = ProgressPattern{DoneChar: "█", PendingChar: "░", PadLeft: " ", PadRight: " "}
	ProgressPlus    = ProgressPattern{DoneChar: "+", PendingChar: " ", PadLeft: "(", PadRight: ")"}
	ProgressHashes  = ProgressPattern{DoneChar: "#", PendingChar: " ", PadLeft: "[", PadRight: "]"}
	ProgressDots    = ProgressPattern{DoneChar: "▪", PendingChar: "▫", PadLeft: " ", PadRight: " "}
	ProgressArrow   = ProgressPattern{DoneChar: "━", PendingChar: "─", PadLeft: " ", PadRight: " "}
	ProgressPipe    = ProgressPattern{DoneChar: "┃", PendingChar: "│", PadLeft: "╟", PadRight: "╢"}
	ProgressShade   = ProgressPattern{DoneChar: "▓", PendingChar: "░", PadLeft: "│", PadRight: "│"}
	ProgressThin    = ProgressPattern{DoneChar: "―", PendingChar: "⋯", PadLeft: " ", PadRight: " "}
)

// ProgressPattern defines the characters used to render the progress bar.
type ProgressPattern struct {
	DoneChar    string
	PendingChar string
	PadLeft     string
	PadRight    string
}

// progress renders an animated progress bar on a single line.
// Construct one with [Progress].
type progress struct {
	cfg            Config
	prefix         string
	label          string
	total          int
	current        int
	width          int
	pattern        ProgressPattern
	stop           bool
	wg             sync.WaitGroup
	mu             sync.Mutex
	lastCompletion int
	lineHeight     int
}

// Progress returns a progress bar builder with sensible defaults.
//
//	pb := termactions.Progress().WithLabel("uploading...").WithTotal(100)
//	pb.Start()
//	for _, f := range files {
//	    upload(f)
//	    pb.Increment() // blocks on the final call until cleanup completes
//	}
//	termactions.Log().Info("all files uploaded") // terminal is guaranteed clean
func Progress() *progress {
	return &progress{
		cfg:     pkgConfig,
		prefix:  "(~)",
		label:   "Loading",
		total:   100,
		width:   40,
		pattern: ProgressDefault,
	}
}

// WithStyles overrides the [StyleMap] for this progress bar.
func (pr *progress) WithStyles(s *StyleMap) *progress {
	pr.cfg.Styles = s
	return pr
}

// WithPrefix overrides the default prefix displayed before the label.
func (pr *progress) WithPrefix(prefix string) *progress {
	pr.prefix = prefix
	return pr
}

// WithLabel sets the label displayed beside the progress bar.
func (pr *progress) WithLabel(label string) *progress {
	pr.label = label
	return pr
}

// WithTotal sets the total number of steps for the progress bar.
func (pr *progress) WithTotal(total int) *progress {
	pr.total = max(1, total)
	return pr
}

// WithWidth sets the maximum width of the bar in characters. Defaults to 40.
func (pr *progress) WithWidth(width int) *progress {
	pr.width = max(1, width)
	return pr
}

// WithPattern sets a custom [ProgressPattern] for the bar characters.
func (pr *progress) WithPattern(p ProgressPattern) *progress {
	pr.pattern = p
	return pr
}

// UpdateLabel changes the progress bar label while it is running.
// Safe to call from any goroutine.
//
//	pb.UpdateLabel("uploading file 3/10...")
func (pr *progress) UpdateLabel(label string) {
	pr.mu.Lock()
	pr.label = label
	pr.mu.Unlock()

	if pr.cfg.Accessible {
		stdOutput.Write([]byte(
			safeStyle(pr.cfg.Styles.ProgressPrefix).Sprint(pr.prefix) + " " +
				safeStyle(pr.cfg.Styles.ProgressLabel).Sprint(label) + "\n"))
	}
}

// Start begins the progress bar render loop in a background goroutine.
// The bar cleans up automatically when the total is reached.
// In accessible mode, prints milestone lines instead of animating.
func (pr *progress) Start() {
	if !pr.cfg.Accessible {
		stdOutput.Write([]byte(ansiHideCursor))
	}

	// Watch for Ctrl+C: restore terminal before exit
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		pr.stop = true
		pr.wg.Wait()
		os.Exit(1)
	}()

	pr.wg.Go(func() {
		if pr.cfg.Accessible {
			for !pr.stop {
				pr.redraw()
				time.Sleep(100 * time.Millisecond)
			}
			pr.redraw()
			return
		}

		defer func() {
			if pr.lineHeight > 1 {
				ansiCursorUp(pr.lineHeight - 1)
			}
			stdOutput.Write([]byte("\r" + ansiClearScreen + ansiShowCursor))
		}()

		for !pr.stop {
			pr.redraw()
			time.Sleep(100 * time.Millisecond)
		}
		pr.redraw()
	})
}

// Increment advances the progress bar by one step.
// Automatically cleans up when the total is reached.
func (pr *progress) Increment() {
	pr.mu.Lock()
	if pr.current < pr.total {
		pr.current++
	}
	done := pr.current == pr.total
	pr.mu.Unlock()

	if done {
		pr.stop = true
		pr.wg.Wait()
	}
}

// Set sets the progress bar to a specific value.
// Automatically cleans up when the total is reached.
func (pr *progress) Set(n int) {
	pr.mu.Lock()
	pr.current = min(max(n, 0), pr.total)
	done := pr.current == pr.total
	pr.mu.Unlock()

	if done {
		pr.stop = true
		pr.wg.Wait()
	}
}

// redraw renders the current progress bar state to the terminal.
func (pr *progress) redraw() {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	// Clamp ratio between 0 and 1
	ratio := float64(pr.current) / float64(pr.total)
	ratio = min(max(ratio, 0), 1)

	// Format percentage padded to 4 chars
	percent := strconv.Itoa(int(ratio * 100))
	for runewidth.StringWidth(percent) < 4 {
		percent = " " + percent
	}
	percent += "%"

	// Determine available width for the bar
	termWidth, _, _ := termSize()
	if termWidth <= 0 {
		termWidth = 80
	}
	fixedWidth := runewidth.StringWidth(pr.prefix + " " + pr.label + " " + pr.pattern.PadLeft + pr.pattern.PadRight + "  " + percent)
	availWidth := max(termWidth-fixedWidth, 0)
	barWidth := min(availWidth, pr.width)

	// Calculate filled and pending segments
	filled := min(int(ratio*float64(barWidth)), barWidth)
	pending := barWidth - filled

	// Accessible mode: print milestone lines
	if pr.cfg.Accessible {
		milestone := int(ratio * 10) // 0-10
		for pr.lastCompletion < milestone {
			pr.lastCompletion++
			pct := strconv.Itoa(pr.lastCompletion * 10)
			stdOutput.Write([]byte(
				safeStyle(pr.cfg.Styles.ProgressPrefix).Sprint(pr.prefix) + " " +
					safeStyle(pr.cfg.Styles.ProgressLabel).Sprint(pr.label) + " [" +
					safeStyle(pr.cfg.Styles.ProgressBarStatus).Sprint(pct+"%") + "]\n"))
		}
		return
	}

	// Build styled bar
	bar := safeStyle(pr.cfg.Styles.ProgressBarPad).Sprint(pr.pattern.PadLeft) +
		safeStyle(pr.cfg.Styles.ProgressBarDone).Sprint(strings.Repeat(pr.pattern.DoneChar, filled)) +
		safeStyle(pr.cfg.Styles.ProgressBarPending).Sprint(strings.Repeat(pr.pattern.PendingChar, pending)) +
		safeStyle(pr.cfg.Styles.ProgressBarPad).Sprint(pr.pattern.PadRight)

	line := safeStyle(pr.cfg.Styles.ProgressPrefix).Sprint(pr.prefix) + " " +
		safeStyle(pr.cfg.Styles.ProgressLabel).Sprint(pr.label) + " " +
		bar +
		safeStyle(pr.cfg.Styles.ProgressBarStatus).Sprint(percent)

	newHeight := physicalLines(stripAnsi(line), termWidth)

	// Move to top of previous frame
	if pr.lineHeight > 1 {
		ansiCursorUp(pr.lineHeight - 1)
	}
	stdOutput.Write([]byte("\r" + ansiClearScreen + line))

	pr.lineHeight = newHeight
}
