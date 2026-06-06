package termactions

import "github.com/fatih/color"

// ==== Log Message ============================================================

// log prints a single styled log line with a level prefix.
// Construct one with [Log].
type log struct {
	cfg    Config
	prefix string
}

// Log returns a builder for printing a single styled log line.
//
//	termactions.Log().Info("server started")
//	termactions.Log().WithPrefix("(done)").Success("deployment complete")
func Log() *log {
	return &log{cfg: pkgConfig}
}

// WithStyles overrides the [StyleMap] for this message.
func (l *log) WithStyles(s *StyleMap) *log {
	l.cfg.Styles = s
	return l
}

// WithPrefix overrides the default level prefix.
func (l *log) WithPrefix(p string) *log {
	l.prefix = p
	return l
}

// Success prints a success message.
func (l *log) Success(msg string) {
	l.render(l.cfg.Styles.LogSuccessPrefix, l.cfg.Styles.LogSuccessLabel, "(✓)", msg)
}

// Debug prints a debug message.
func (l *log) Debug(msg string) {
	l.render(l.cfg.Styles.LogDebugPrefix, l.cfg.Styles.LogDebugLabel, "(~)", msg)
}

// Info prints an info message.
func (l *log) Info(msg string) {
	l.render(l.cfg.Styles.LogInfoPrefix, l.cfg.Styles.LogInfoLabel, "(i)", msg)
}

// Warn prints a warning message.
func (l *log) Warn(msg string) {
	l.render(l.cfg.Styles.LogWarnPrefix, l.cfg.Styles.LogWarnLabel, "(!)", msg)
}

// Error prints an error message.
func (l *log) Error(msg string) {
	l.render(l.cfg.Styles.LogErrorPrefix, l.cfg.Styles.LogErrorLabel, "(✗)", msg)
}

// render prints a level prefixed log line.
func (l *log) render(pfxStyle, labelStyle *color.Color, defaultPfx, msg string) {
	pfx := safeStyle(pfxStyle).Sprint(pick(l.prefix, defaultPfx))
	label := safeStyle(labelStyle).Sprint(msg)
	stdOutput.Write([]byte(pfx + " " + label + "\n"))
}

// ==== Log Group ==============================================================

// logGroup prints a styled title line followed by indented body lines.
// Construct one with [LogGroup].
type logGroup struct {
	cfg    Config
	prefix string
}

// LogGroup returns a builder for printing a styled title with indented body lines.
//
//	termactions.LogGroup().Info("config loaded", "host: localhost", "port: 8080")
//	termactions.LogGroup().WithPrefix("DONE:").Success("deploy finished", "3 services restarted")
func LogGroup() *logGroup {
	return &logGroup{cfg: pkgConfig}
}

// WithStyles overrides the [StyleMap] for this group.
func (l *logGroup) WithStyles(s *StyleMap) *logGroup {
	l.cfg.Styles = s
	return l
}

// WithPrefix overrides the default level prefix.
func (l *logGroup) WithPrefix(p string) *logGroup {
	l.prefix = p
	return l
}

// Success prints a success group.
func (l *logGroup) Success(title string, msgs ...string) {
	l.render(l.cfg.Styles.LogSuccessPrefix, l.cfg.Styles.LogSuccessLabel, "SUCCESS:", title, msgs...)
}

// Debug prints a debug group.
func (l *logGroup) Debug(title string, msgs ...string) {
	l.render(l.cfg.Styles.LogDebugPrefix, l.cfg.Styles.LogDebugLabel, "DEBUG:", title, msgs...)
}

// Info prints an info group.
func (l *logGroup) Info(title string, msgs ...string) {
	l.render(l.cfg.Styles.LogInfoPrefix, l.cfg.Styles.LogInfoLabel, "INFO:", title, msgs...)
}

// Warn prints a warning group.
func (l *logGroup) Warn(title string, msgs ...string) {
	l.render(l.cfg.Styles.LogWarnPrefix, l.cfg.Styles.LogWarnLabel, "WARN:", title, msgs...)
}

// Error prints an error group.
func (l *logGroup) Error(title string, msgs ...string) {
	l.render(l.cfg.Styles.LogErrorPrefix, l.cfg.Styles.LogErrorLabel, "ERROR:", title, msgs...)
}

// render prints a level prefixed title line followed by indented body lines.
func (l *logGroup) render(pfxStyle, labelStyle *color.Color, defaultPfx, title string, msgs ...string) {
	pfx := safeStyle(pfxStyle).Sprint(pick(l.prefix, defaultPfx))
	titleStr := safeStyle(labelStyle).Sprint(title)
	stdOutput.Write([]byte(pfx + " " + titleStr + "\n"))
	for _, msg := range msgs {
		stdOutput.Write([]byte("  " + safeStyle(l.cfg.Styles.LogGroupBody).Sprint(msg) + "\n"))
	}
}

// ==== Log Block ==============================================================

// logBlock prints a colored title line, a blank line, body lines, and a trailing blank line.
// Construct one with [LogBlock].
type logBlock struct {
	cfg Config
}

// LogBlock returns a builder for printing a colored title with spaced body lines.
//
//	termactions.LogBlock().Info("migration complete", "3 records updated", "0 errors")
//	termactions.LogBlock().Warn("config missing", "falling back to defaults", "run init to configure")
func LogBlock() *logBlock {
	return &logBlock{cfg: pkgConfig}
}

// WithStyles overrides the [StyleMap] for this block.
func (l *logBlock) WithStyles(s *StyleMap) *logBlock {
	l.cfg.Styles = s
	return l
}

// Success prints a success block.
func (l *logBlock) Success(title string, msgs ...string) {
	l.render(l.cfg.Styles.LogSuccessPrefix, title, msgs...)
}

// Debug prints a debug block.
func (l *logBlock) Debug(title string, msgs ...string) {
	l.render(l.cfg.Styles.LogDebugPrefix, title, msgs...)
}

// Info prints an info block.
func (l *logBlock) Info(title string, msgs ...string) {
	l.render(l.cfg.Styles.LogInfoPrefix, title, msgs...)
}

// Warn prints a warning block.
func (l *logBlock) Warn(title string, msgs ...string) {
	l.render(l.cfg.Styles.LogWarnPrefix, title, msgs...)
}

// Error prints an error block.
func (l *logBlock) Error(title string, msgs ...string) {
	l.render(l.cfg.Styles.LogErrorPrefix, title, msgs...)
}

// render prints a level styled title line with spaced body lines.
func (l *logBlock) render(titleStyle *color.Color, title string, msgs ...string) {
	stdOutput.Write([]byte(safeStyle(titleStyle).Sprint(title) + "\n\n"))
	for _, msg := range msgs {
		stdOutput.Write([]byte(safeStyle(l.cfg.Styles.LogGroupBody).Sprint(msg) + "\n"))
	}
	stdOutput.Write([]byte("\n"))
}
