package termactions

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Spinner frame pattern presets.
var (
	SpinnerDefault  = []string{"(⠋)", "(⠙)", "(⠹)", "(⠸)", "(⠼)", "(⠴)", "(⠦)", "(⠧)", "(⠇)", "(⠏)"}
	SpinnerDots     = []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	SpinnerDotsMini = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	SpinnerCircles  = []string{"◐", "◓", "◑", "◒"}
	SpinnerSquares  = []string{"▖", "▌", "▘", "▀", "▝", "▐", "▗", "▄"}
	SpinnerLine     = []string{"-", "\\", "|", "/"}
	SpinnerPipes    = []string{"╾", "│", "╸", "┤", "├", "└", "┴", "┬", "┐", "┘"}
	SpinnerMoons    = []string{"🌑", "🌒", "🌓", "🌔", "🌕", "🌖", "🌗", "🌘"}
	SpinnerBounce   = []string{"⠁", "⠂", "⠄", "⡀", "⢀", "⠠", "⠐", "⠈"}
	SpinnerArrows   = []string{"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"}
	SpinnerGrow     = []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█", "▇", "▆", "▅", "▄", "▃", "▂"}
	SpinnerToggle   = []string{"⊶", "⊷"}
	SpinnerArc      = []string{"◜", "◠", "◝", "◞", "◡", "◟"}
	SpinnerBall     = []string{"( ●    )", "(  ●   )", "(   ●  )", "(    ● )", "(     ●)", "(    ● )", "(   ●  )", "(  ●   )", "( ●    )", "(●     )"}
)

// spinner renders an animated spinner on a single line.
// Construct one with [Spinner].
type spinner struct {
	cfg      Config
	frames   []string
	label    string
	interval time.Duration
	stop     bool
	mu       sync.Mutex
	wg       sync.WaitGroup
}

// Spinner returns a spinner builder with sensible defaults.
//
//	sp := termactions.Spinner().WithLabel("performing action...")
//	sp.Start()
//	// ... do work ...
//	sp.UpdateLabel("finishing up...")
//	// ... more work ...
//	sp.Stop()
func Spinner() *spinner {
	return &spinner{
		cfg:      pkgConfig,
		frames:   SpinnerDefault,
		label:    "Loading",
		interval: 100 * time.Millisecond,
	}
}

// WithStyles overrides the [StyleMap] for this spinner.
func (sp *spinner) WithStyles(s *StyleMap) *spinner {
	sp.cfg.Styles = s
	return sp
}

// WithFrames sets a custom frame pattern for the spinner animation.
func (sp *spinner) WithFrames(frames []string) *spinner {
	sp.frames = frames
	return sp
}

// WithLabel sets the label displayed beside the spinner frame.
func (sp *spinner) WithLabel(label string) *spinner {
	sp.label = label
	return sp
}

// WithInterval sets the frame animation interval. Defaults to 100ms.
func (sp *spinner) WithInterval(d time.Duration) *spinner {
	sp.interval = d
	return sp
}

// UpdateLabel changes the spinner label while the animation is running.
// Safe to call from any goroutine.
//
//	sp.UpdateLabel("processing 3/10...")
func (sp *spinner) UpdateLabel(label string) {
	sp.mu.Lock()
	sp.label = label
	sp.mu.Unlock()

	if sp.cfg.Accessible {
		stdOutput.Write([]byte(sp.frames[0] + " " + label + "\n"))
	}
}

// Start begins the spinner animation in a background goroutine.
// In accessible mode, prints a single static line instead of animating.
func (sp *spinner) Start() {
	if sp.cfg.Accessible {
		stdOutput.Write([]byte(sp.frames[0] + " " + sp.label + "\n"))
		return
	}

	stdOutput.Write([]byte(ansiHideCursor))

	// Watch for Ctrl+C & restore terminal before exit
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		sp.Stop()
		os.Exit(1)
	}()

	sp.wg.Go(func() {
		lineHeight := 0
		i := 0

		defer func() {
			if lineHeight > 1 {
				ansiCursorUp(lineHeight - 1)
			}
			stdOutput.Write([]byte("\r" + ansiClearScreen + ansiShowCursor))
		}()

		for !sp.stop {
			sp.mu.Lock()
			label := sp.label
			sp.mu.Unlock()

			frame := safeStyle(sp.cfg.Styles.SpinnerPrefix).Sprint(sp.frames[i%len(sp.frames)])
			styledLabel := safeStyle(sp.cfg.Styles.SpinnerLabel).Sprint(label)
			line := frame + " " + styledLabel

			termW, _, _ := termSize()
			newHeight := physicalLines(stripAnsi(line), termW)

			// Move to top of previous frame
			if lineHeight > 1 {
				ansiCursorUp(lineHeight - 1)
			}
			stdOutput.Write([]byte("\r" + ansiClearScreen + line))

			lineHeight = newHeight
			i++
			time.Sleep(sp.interval)
		}
	})
}

// Stop halts the spinner and clears the spinner line.
// Safe to call multiple times.
func (sp *spinner) Stop() {
	if sp.cfg.Accessible || sp.stop {
		return
	}
	sp.stop = true
	sp.wg.Wait()
}
