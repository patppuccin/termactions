package termactions

import "github.com/fatih/color"

// Config holds package-level configuration for all Termactions components.
// Set once at program startup using [Configure].
type Config struct {
	// NoColor disables all color output regardless of terminal capability.
	// Note: fatih/color automatically respects the NO_COLOR environment
	// variable and non-TTY output (pipes, redirects). This field is only
	// needed to disable color programmatically at runtime.
	NoColor bool

	// Accessible disables cursor movement and ANSI positioning sequences,
	// printing output linearly instead. Useful for screen readers, CI
	// pipelines, and plain or piped terminal environments.
	Accessible bool

	// Styles sets the [StyleMap] used by all Termactions components.
	// Defaults to [NewStyles] if not set.
	Styles *StyleMap
}

// pkgConfig holds the active package-level configuration.
var pkgConfig = Config{
	Styles: NewStyles(),
}

// Configure sets package-level defaults for all Termactions components.
// Call this once at program startup, before using any Termactions functions.
//
//	termactions.Configure(termactions.Config{
//	    Styles:     myStyles,
//	    Accessible: true,
//	})
func Configure(c Config) {
	if c.NoColor {
		color.NoColor = true
	}
	if c.Accessible {
		pkgConfig.Accessible = true
	}
	if c.Styles != nil {
		pkgConfig.Styles = c.Styles
	}
}
