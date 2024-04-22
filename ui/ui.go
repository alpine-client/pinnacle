package ui

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"github.com/alpine-client/pinnacle/sentry"
	"image"
	"image/color"
	"image/png"
	"sync"

	"github.com/AllenDang/giu"
	"github.com/ncruces/zenity"
)

const (
	LogoSize     float32 = 80
	WindowWidth  int     = 377
	WindowHeight int     = 144
	TotalSteps   float32 = 430
)

var (
	steps  int
	logoI  *image.RGBA
	mutex  sync.Mutex
	window *giu.MasterWindow
	dialog zenity.ProgressDialog
)

func UpdateProgress(i int, msg ...string) {
	mutex.Lock()
	steps += i
	if window != nil {
		giu.Update()
	}
	mutex.Unlock()

	if dialog != nil && len(msg) != 0 {
		_ = dialog.Text(msg[0])
	}
}

func ReadProgress() float32 {
	mutex.Lock()
	p := steps
	mutex.Unlock()
	return float32(p) / TotalSteps
}

func Setup(ctx context.Context, fs embed.FS) {
	defer func() {
		if r := recover(); r != nil {
			sentry.Breadcrumb(ctx, fmt.Sprintf("recovered panic: %v", r), sentry.LevelWarning)
			window = nil
		}
	}()

	window = giu.NewMasterWindow(
		"Alpine Client Updater",
		WindowWidth, WindowHeight,
		giu.MasterWindowFlagsFrameless|giu.MasterWindowFlagsNotResizable|giu.MasterWindowFlagsTransparent,
	)
	if window == nil {
		return
	}
	window.SetBgColor(color.Transparent)

	icon, err := loadImage(fs, "assets/icon.png")
	if err != nil {
		panic(err)
	}
	window.SetIcon([]image.Image{icon})

	logoI, err = loadImage(fs, "assets/logo.png")
	if err != nil {
		panic(err)
	}
}

func defaultUI() {
	SetupStyle()
	giu.SingleWindow().Layout(
		giu.Align(giu.AlignCenter).To(
			giu.Dummy(0, scaleDivider(6)),
			giu.ImageWithRgba(logoI).Size(LogoSize, LogoSize),
			giu.Dummy(0, scaleDivider(6)),
			giu.ProgressBar(ReadProgress()).Size(scaleValue(WindowWidth)*0.75, scaleValue(5)),
		),
	)
	PopStyle()
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
	if window != nil {
		window.Run(defaultUI)
	} else {
		runZenity()
	}
}

func Close() {
	if window != nil {
		window.SetShouldClose(true)
		window = nil
	}
	if dialog != nil {
		_ = dialog.Close()
	}
}

func scaleDivider(value float32) float32 {
	scale := giu.Context.GetPlatform().GetContentScale()
	if scale > 1.0 {
		value *= 2
	}
	return value * scale
}

func scaleValue(value int) float32 {
	scale := giu.Context.GetPlatform().GetContentScale()
	return float32(value) * scale
}

func loadImage(assets embed.FS, path string) (*image.RGBA, error) {
	data, err := assets.ReadFile(path)
	if err != nil {
		return nil, err
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	return giu.ImageToRgba(img), nil
}
