package ui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// LogEntry holds a single parsed log record for display.
type LogEntry struct {
	Time    time.Time
	Level   slog.Level
	Message string
	Attrs   []slog.Attr
}

// WidgetHandler is an slog.Handler that forwards records to a LogWidget.
type WidgetHandler struct {
	widget *LogWidget
	level  slog.Level
	attrs  []slog.Attr
	groups []string
}

func NewWidgetHandler(w *LogWidget, level slog.Level) *WidgetHandler {
	return &WidgetHandler{widget: w, level: level}
}

func (h *WidgetHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *WidgetHandler) Handle(_ context.Context, r slog.Record) error {
	entry := LogEntry{
		Time:    r.Time,
		Level:   r.Level,
		Message: r.Message,
		Attrs:   make([]slog.Attr, 0, len(h.attrs)+r.NumAttrs()),
	}
	entry.Attrs = append(entry.Attrs, h.attrs...)
	r.Attrs(func(a slog.Attr) bool {
		entry.Attrs = append(entry.Attrs, a)
		return true
	})
	h.widget.Append(entry)
	return nil
}

func (h *WidgetHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	combined := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(combined, h.attrs)
	copy(combined[len(h.attrs):], attrs)
	return &WidgetHandler{widget: h.widget, level: h.level, attrs: combined, groups: h.groups}
}

func (h *WidgetHandler) WithGroup(name string) slog.Handler {
	groups := append(append([]string{}, h.groups...), name)
	return &WidgetHandler{widget: h.widget, level: h.level, attrs: h.attrs, groups: groups}
}

// FormatEntry produces a single-line string for an entry.
func FormatEntry(e LogEntry) string {
	ts := e.Time.Format("15:04:05.000")
	level := levelTag(e.Level)
	var sb strings.Builder
	fmt.Fprintf(&sb, "[%s] %s  %s", ts, level, e.Message)
	for _, a := range e.Attrs {
		fmt.Fprintf(&sb, "  %s=%v", a.Key, a.Value)
	}
	return sb.String()
}

func levelTag(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return "ERR "
	case l >= slog.LevelWarn:
		return "WARN"
	case l >= slog.LevelInfo:
		return "INFO"
	default:
		return "DBG "
	}
}
