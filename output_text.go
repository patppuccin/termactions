package termactions

import (
	"strings"

	"github.com/fatih/color"
)

// ==== Message ================================================================

// message prints unprefixed lines in a chosen style.
// Construct one with [Message].
type message struct {
	cfg Config
}

// Message returns a builder for printing unprefixed lines, one per argument.
//
//	termactions.Message().Info("Create master password")
//	termactions.Message().Neutral("Stored at /path/to/vault", "Keep it safe")
func Message() *message {
	return &message{cfg: pkgConfig}
}

// WithStyles overrides the [StyleMap] for this message.
func (m *message) WithStyles(s *StyleMap) *message {
	m.cfg.Styles = s
	return m
}

// Success prints success lines.
func (m *message) Success(msgs ...string) {
	m.render(m.cfg.Styles.MsgSuccessPrefix, msgs)
}

// Debug prints debug lines.
func (m *message) Debug(msgs ...string) {
	m.render(m.cfg.Styles.MsgDebugPrefix, msgs)
}

// Info prints info lines.
func (m *message) Info(msgs ...string) {
	m.render(m.cfg.Styles.MsgInfoPrefix, msgs)
}

// Warn prints warning lines.
func (m *message) Warn(msgs ...string) {
	m.render(m.cfg.Styles.MsgWarnPrefix, msgs)
}

// Error prints error lines.
func (m *message) Error(msgs ...string) {
	m.render(m.cfg.Styles.MsgErrorPrefix, msgs)
}

// Neutral prints lines in the neutral style.
func (m *message) Neutral(msgs ...string) {
	m.render(m.cfg.Styles.MsgNeutral, msgs)
}

// Muted prints lines in the muted style.
func (m *message) Muted(msgs ...string) {
	m.render(m.cfg.Styles.MsgMuted, msgs)
}

// render prints each message on its own line in the given style.
func (m *message) render(style *color.Color, msgs []string) {
	var b strings.Builder
	for _, msg := range msgs {
		b.WriteString(safeStyle(style).Sprint(msg))
		b.WriteByte('\n')
	}
	stdOutput.Write([]byte(b.String()))
}

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
	l.render(l.cfg.Styles.MsgSuccessPrefix, l.cfg.Styles.MsgSuccessLabel, "(✓)", msg)
}

// Debug prints a debug message.
func (l *log) Debug(msg string) {
	l.render(l.cfg.Styles.MsgDebugPrefix, l.cfg.Styles.MsgDebugLabel, "(~)", msg)
}

// Info prints an info message.
func (l *log) Info(msg string) {
	l.render(l.cfg.Styles.MsgInfoPrefix, l.cfg.Styles.MsgInfoLabel, "(i)", msg)
}

// Warn prints a warning message.
func (l *log) Warn(msg string) {
	l.render(l.cfg.Styles.MsgWarnPrefix, l.cfg.Styles.MsgWarnLabel, "(!)", msg)
}

// Error prints an error message.
func (l *log) Error(msg string) {
	l.render(l.cfg.Styles.MsgErrorPrefix, l.cfg.Styles.MsgErrorLabel, "(✗)", msg)
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
	l.render(l.cfg.Styles.MsgSuccessPrefix, l.cfg.Styles.MsgSuccessLabel, "SUCCESS:", title, msgs...)
}

// Debug prints a debug group.
func (l *logGroup) Debug(title string, msgs ...string) {
	l.render(l.cfg.Styles.MsgDebugPrefix, l.cfg.Styles.MsgDebugLabel, "DEBUG:", title, msgs...)
}

// Info prints an info group.
func (l *logGroup) Info(title string, msgs ...string) {
	l.render(l.cfg.Styles.MsgInfoPrefix, l.cfg.Styles.MsgInfoLabel, "INFO:", title, msgs...)
}

// Warn prints a warning group.
func (l *logGroup) Warn(title string, msgs ...string) {
	l.render(l.cfg.Styles.MsgWarnPrefix, l.cfg.Styles.MsgWarnLabel, "WARN:", title, msgs...)
}

// Error prints an error group.
func (l *logGroup) Error(title string, msgs ...string) {
	l.render(l.cfg.Styles.MsgErrorPrefix, l.cfg.Styles.MsgErrorLabel, "ERROR:", title, msgs...)
}

// render prints a level prefixed title line followed by indented body lines.
func (l *logGroup) render(pfxStyle, labelStyle *color.Color, defaultPfx, title string, msgs ...string) {
	pfx := safeStyle(pfxStyle).Sprint(pick(l.prefix, defaultPfx))
	titleStr := safeStyle(labelStyle).Sprint(title)
	stdOutput.Write([]byte(pfx + " " + titleStr + "\n"))
	for _, msg := range msgs {
		stdOutput.Write([]byte("  " + safeStyle(l.cfg.Styles.MsgNeutral).Sprint(msg) + "\n"))
	}
}
