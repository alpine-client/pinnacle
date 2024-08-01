package ui

import (
	"bytes"
	"embed"
	"image"
	"image/png"
	"os"

	"github.com/ncruces/zenity"
)

const (
	// LogoSize     float32 = 80
	WindowWidth  int = 377
	WindowHeight int = 144
)

var (
	logoI  *os.File
	dialog zenity.ProgressDialog
	tasks  []*ProgressiveTask
)

func Render() {
	runZenity()
}

func Close() {
	if dialog != nil {
		_ = dialog.Close()
	}
	if logoI != nil {
		_ = os.Remove(logoI.Name())
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

func LoadImage(assets embed.FS, path string) (image.Image, error) {
	data, err := assets.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return png.Decode(bytes.NewReader(data))
}
