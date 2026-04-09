// Package log configures and exposes the application-wide structured logger.
//
// The logger is backed by [log/slog]. Call [Init] once at startup (wired into
// the root cobra PersistentPreRunE) before any command handler runs.
//
// Convention:
//   - [slog.Debug] – internal operation traces; only emitted with --verbose.
//   - [slog.Info]  – significant lifecycle events always written to stderr.
//   - fmt.Print*   – user-facing command output written to stdout; never replaced.
package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Init configures the global slog logger.
//
//   - verbose=false → Info level, compact format (no timestamps), stderr.
//   - verbose=true  → Debug level, full text format with timestamps, stderr.
func Init(verbose bool) {
	var h slog.Handler
	if verbose {
		h = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		h = newCompactHandler(os.Stderr, slog.LevelInfo)
	}
	slog.SetDefault(slog.New(h))
}

// compactHandler writes log records as a single line without timestamps:
//
//	INFO  QLT Startup...
//	DEBUG Executing init command...
type compactHandler struct {
	w     io.Writer
	level slog.Level
}

func newCompactHandler(w io.Writer, level slog.Level) *compactHandler {
	return &compactHandler{w: w, level: level}
}

func (h *compactHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *compactHandler) Handle(_ context.Context, r slog.Record) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-5s %s", r.Level.String(), r.Message))
	r.Attrs(func(a slog.Attr) bool {
		sb.WriteString(fmt.Sprintf(" %s=%v", a.Key, a.Value))
		return true
	})
	sb.WriteByte('\n')
	_, err := io.WriteString(h.w, sb.String())
	return err
}

func (h *compactHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Compact handler is stateless; attrs are written inline in Handle.
	return h
}

func (h *compactHandler) WithGroup(name string) slog.Handler {
	return h
}
