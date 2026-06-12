package ui

import (
	"fmt"
	"image/color"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/SwissOpenEM/Ingestor/internal/transfertask"
)

// --- Stub types (replace with your real imports) ---

type TransferMethod string

const (
	TransferMethodS3  TransferMethod = "S3"
	TransferMethodSCP TransferMethod = "SCP"
)

type DatasetFolder struct {
	Path string
}

type TaskStatus string

const (
	StatusPending   TaskStatus = "Pending"
	StatusUploading TaskStatus = "Uploading"
	StatusDone      TaskStatus = "Done"
	StatusFailed    TaskStatus = "Failed"
)

type TaskDetails struct {
	Status       TaskStatus
	Progress     float64 // 0.0 – 1.0
	BytesDone    int64
	BytesTotal   int64
	ErrorMessage string
}

// --- Task row widget ---

type taskRow struct {
	widget.BaseWidget
	task   *transfertask.TransferTask
	folder *widget.Label
	// method   *widget.Label
	status   *widget.Label
	progress *widget.ProgressBar
	cancel   *widget.Button
}

func newTaskRow(task *transfertask.TransferTask) *taskRow {
	r := &taskRow{task: task}

	r.folder = widget.NewLabelWithStyle(
		task.DatasetFolder.FolderPath,
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)
	// r.method = widget.NewLabel(string(task.TransferMethod))
	r.status = widget.NewLabel("Pending")
	r.progress = widget.NewProgressBar()
	r.cancel = widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		if task.Cancel != nil {
			task.Cancel()
		}
	})
	r.cancel.Hide()

	r.ExtendBaseWidget(r)
	return r
}

func (r *taskRow) Refresh() {
	d := r.task.GetDetails()

	r.folder.SetText(r.task.DatasetFolder.FolderPath)
	r.progress.SetValue(float64(d.BytesTransferred) / float64(d.BytesTotal))

	switch d.Status {
	case transfertask.Finished:
		r.status.SetText("✓ Done")
		r.status.Importance = widget.SuccessImportance
		r.cancel.Disable()
	case transfertask.Failed:
		r.status.SetText("✗ " + d.Message)
		r.status.Importance = widget.DangerImportance
		r.cancel.Disable()
	case transfertask.Transferring:
		mb := float64(d.BytesTransferred) / 1024 / 1024
		total := float64(d.BytesTotal) / 1024 / 1024
		r.status.SetText(fmt.Sprintf("Uploading %.1f / %.1f MB", mb, total))
		r.status.Importance = widget.MediumImportance
	default:
		r.status.SetText(d.Status.ToStr())
	}

	r.BaseWidget.Refresh()
}

func (r *taskRow) CreateRenderer() fyne.WidgetRenderer {
	top := container.NewBorder(nil, nil, nil, r.cancel,
		container.NewGridWithColumns(1, r.folder),
	)
	bottom := container.NewBorder(nil, nil, r.status, nil, r.progress)
	card := container.NewVBox(top, bottom)
	return widget.NewSimpleRenderer(card)
}

// --- Task list ---

type TaskListUI struct {
	mu     sync.RWMutex
	tasks  []*transfertask.TransferTask
	list   *fyne.Container
	scroll *container.Scroll
}

func NewTaskListUI(tasks []*transfertask.TransferTask) *TaskListUI {
	ui := &TaskListUI{}
	ui.list = container.NewVBox()
	ui.scroll = container.NewVScroll(ui.list)

	for _, t := range tasks {
		ui.addRow(t)
	}

	return ui
}

func (ui *TaskListUI) addRow(t *transfertask.TransferTask) {
	row := newTaskRow(t)
	ui.tasks = append(ui.tasks, t)
	ui.list.Add(row)
	ui.list.Add(widget.NewSeparator())
}

func (ui *TaskListUI) Refresh() {
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	for _, obj := range ui.list.Objects {
		if row, ok := obj.(*taskRow); ok {
			row.Refresh()
		}
	}
}

func (ui *TaskListUI) Container() fyne.CanvasObject {
	return container.NewVScroll(ui.list)
}

func (ui *TaskListUI) SetTasks(tasks []transfertask.TransferTask) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	ui.tasks = nil
	ui.list.Objects = nil
	for _, t := range tasks {
		ui.addRow(&t)
	}
	ui.list.Refresh()
}

// --- Main ---

// Custom dark theme mimicking PSI Discovery portal
type PsiTheme struct{}

func (p PsiTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.RGBA{39, 170, 225, 1}
	case theme.ColorNameForeground:
		return color.Color(color.White)
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 0x38, G: 0xbd, B: 0xb3, A: 0xff}
	case theme.ColorNameButton:
		return color.NRGBA{R: 0x2a, G: 0x2f, B: 0x45, A: 0xff}
	case theme.ColorNameDisabledButton:
		return color.NRGBA{R: 0x2a, G: 0x2f, B: 0x45, A: 0x88}
	case theme.ColorNameSeparator:
		return color.NRGBA{R: 0x38, G: 0x3e, B: 0x56, A: 0xff} // subtle divider
	case theme.ColorNameScrollBar:
		return color.NRGBA{R: 0x38, G: 0xbd, B: 0xb3, A: 0x88}
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (p PsiTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (p PsiTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (p PsiTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 8
	case theme.SizeNameInnerPadding:
		return 6
	case theme.SizeNameText:
		return 13
	case theme.SizeNameHeadingText:
		return 15
	}
	return theme.DefaultTheme().Size(name)
}
