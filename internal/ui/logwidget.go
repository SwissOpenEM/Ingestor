package ui

import (
	"image/color"
	"log/slog"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const maxEntries = 1000

// LogWidget is a scrollable list widget that shows slog records.
type LogWidget struct {
	widget.BaseWidget

	mu      sync.Mutex
	entries []LogEntry

	list   *widget.List
	scroll *container.Scroll
}

func NewLogWidget() *LogWidget {
	lw := &LogWidget{}
	lw.ExtendBaseWidget(lw)

	lw.list = widget.NewList(
		func() int {
			lw.mu.Lock()
			defer lw.mu.Unlock()
			return len(lw.entries)
		},
		func() fyne.CanvasObject {
			lbl := canvas.NewText("", color.White)
			lbl.TextStyle = fyne.TextStyle{Monospace: true}
			lbl.TextSize = 12
			return lbl
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			lw.mu.Lock()
			if id >= len(lw.entries) {
				lw.mu.Unlock()
				return
			}
			entry := lw.entries[id]
			lw.mu.Unlock()

			lbl := obj.(*canvas.Text)
			lbl.Text = FormatEntry(entry)
			lbl.Color = levelColor(entry.Level)
			go func() {
				fyne.Do(
					func() {
						lbl.Refresh()
					})
			}()
		})

	lw.list.HideSeparators = true
	lw.scroll = container.NewScroll(lw.list)
	return lw
}

// Append adds a new log entry and refreshes the list.
func (lw *LogWidget) Append(e LogEntry) {
	lw.mu.Lock()
	lw.entries = append(lw.entries, e)
	if len(lw.entries) > maxEntries {
		lw.entries = lw.entries[len(lw.entries)-maxEntries:]
	}
	count := len(lw.entries)
	lw.mu.Unlock()

	go func() {
		fyne.Do(
			func() {
				lw.list.Refresh()
				lw.list.ScrollTo(count - 1)
			})
	}()
}

// Clear removes all entries.
func (lw *LogWidget) Clear() {
	lw.mu.Lock()
	lw.entries = lw.entries[:0]
	lw.mu.Unlock()
	lw.list.Refresh()
}

// CreateRenderer implements fyne.Widget.
func (lw *LogWidget) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(lw.scroll)
}

// MinSize returns a sensible default.
func (lw *LogWidget) MinSize() fyne.Size {
	return fyne.NewSize(600, 300)
}

func levelColor(l slog.Level) color.Color {
	switch {
	case l >= slog.LevelError:
		return color.NRGBA{R: 220, G: 0, B: 37, A: 255}
	case l >= slog.LevelWarn:
		return color.NRGBA{R: 255, G: 200, B: 50, A: 255}
	case l >= slog.LevelInfo:
		return color.Color(theme.Color(theme.ColorNameForeground))
	default:
		return color.NRGBA{R: 67, G: 70, B: 75, A: 255}
	}
}
