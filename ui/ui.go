package ui

import (
	"fmt"
	"log"
	"math"

	"github.com/ncruces/zenity"
)

const (
	WindowTitle  string = "Alpine Client"
	WindowWidth  int    = 377
	WindowHeight int    = 144
	MaxProgress  int    = 1000 // 1000 is used over 100 to make bar smoother
)

var (
	dialog zenity.ProgressDialog
	tasks  []*ProgressiveTask
)

type ProgressiveTask struct {
	label    string
	progress int
}

func NewProgressTask(label string) *ProgressiveTask {
	pt := &ProgressiveTask{
		label: label,
	}
	tasks = append(tasks, pt)
	if dialog != nil {
		_ = dialog.Text(label)
	}
	return pt
}

func (pt *ProgressiveTask) UpdateProgress(v float64, label ...string) {
	if dialog == nil {
		return
	}

	if len(label) > 0 {
		pt.label = label[0]
	}

	progress := int(math.Floor(v * float64(MaxProgress)))
	if progress < MaxProgress {
		pt.progress = progress
	}

	_ = dialog.Text(fmt.Sprintf("%s %d%%", pt.label, pt.progress/10))
	_ = dialog.Value(pt.progress)
}

func Render() {
	var err error
	dialog, err = zenity.Progress(
		zenity.NoCancel(),
		zenity.Title(WindowTitle),
		zenity.MaxValue(MaxProgress),
		zenity.Width(uint(WindowWidth)),
		zenity.Height(uint(WindowHeight)),
	)
	if err != nil {
		log.Printf("[ERROR] failed to render progress bar: %v", err)
		dialog = nil
	}
}

func Close() {
	if dialog != nil {
		_ = dialog.Close()
	}
}
