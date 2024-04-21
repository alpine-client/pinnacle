package ui

import (
	"bytes"
	"context"
	"embed"
	"image"
	"image/color"
	"image/png"
	"sync"

	"github.com/AllenDang/giu"
	"github.com/alpine-client/pinnacle/sentry"
	"github.com/ncruces/zenity"
)

const (
	LogoSize     float32 = 80
	WindowWidth  int     = 377
	WindowHeight int     = 144
	TotalSteps   float32 = 430
)

var (
	steps int
	logoI *image.RGBA
	mutex sync.Mutex
)

func UpdateProgress(i int) {
	mutex.Lock()
	steps += i
	// giu.Update()
	mutex.Unlock()
}

func ReadProgress() float32 {
	mutex.Lock()
	p := steps
	mutex.Unlock()
	return float32(p) / TotalSteps
}

func NewWindow(ctx context.Context, fs embed.FS) *giu.MasterWindow {
	defer sentry.Recover(ctx)

	window := giu.NewMasterWindow(
		"Alpine Client Updater",
		WindowWidth, WindowHeight,
		giu.MasterWindowFlagsFrameless|giu.MasterWindowFlagsNotResizable|giu.MasterWindowFlagsTransparent,
	)
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

	return window
}

func NewZenityWindow(ctx context.Context) zenity.ProgressDialog {
	dialog, err := zenity.Progress(
		zenity.Title("Starting Alpine Client"),
		zenity.AutoClose(),
		zenity.NoCancel(),
		zenity.Pulsate(),
	)
	sentry.CaptureErrExit(ctx, err)
	return dialog
}

func RenderDefault() {
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
