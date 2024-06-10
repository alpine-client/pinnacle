package ui

import (
	"github.com/ncruces/zenity"
)

const (
	WindowWidth  int     = 377
	WindowHeight int     = 144
	TotalSteps   float32 = 430
)

var dialog zenity.ProgressDialog

func UpdateProgress(_ int, msg ...string) {
	if dialog != nil && len(msg) != 0 {
		_ = dialog.Text(msg[0])
	}
}

func runZenity() {
	var err error
	dialog, err = zenity.Progress(
		zenity.Title("Starting Alpine Client"),
		zenity.NoCancel(),
		zenity.Pulsate(),
		zenity.Width(uint(WindowWidth)),
		zenity.Height(uint(WindowHeight)),
	)
	if err != nil {
		dialog = nil
	}
}

func Render() {
	runZenity()
}

func Close() {
	if dialog != nil {
		_ = dialog.Close()
	}
}
