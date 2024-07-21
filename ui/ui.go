package ui

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
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
	steps  int
	logoI  *os.File
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

	icon, err := loadRGBAImage(fs, "assets/icon.png")
	if err != nil {
		panic(err)
	}
	window.SetIcon(icon)

	logoI, err = loadTempImage(fs, "assets/logo.png")
	if err != nil {
		panic(err)
	}
}

func defaultUI() {
	SetupStyle()
	giu.SingleWindow().Layout(
		giu.Align(giu.AlignCenter).To(
			giu.Dummy(0, scaleDivider(6)),
			giu.ImageWithFile(logoI.Name()).Size(LogoSize, LogoSize),
			giu.Dummy(0, scaleDivider(6)),
			giu.ProgressBar(ReadProgress()).Size(scaleValueX(WindowWidth)*0.75, scaleValueY(5)),
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
	if logoI != nil {
		_ = os.Remove(logoI.Name())
	}
}

func scaleDivider(value float32) float32 {
	_, yScale := giu.Context.Backend().ContentScale()
	if yScale > 1.0 {
		value *= 2
	}
	return value * yScale
}

func scaleValueY(value int) float32 {
	_, yScale := giu.Context.Backend().ContentScale()
	return float32(value) * yScale
}

func scaleValueX(value int) float32 {
	xScale, _ := giu.Context.Backend().ContentScale()
	return float32(value) * xScale
}

func loadRGBAImage(assets embed.FS, path string) (*image.RGBA, error) {
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

func loadTempImage(assets embed.FS, path string) (*os.File, error) {
	tempFile, err := os.CreateTemp("", "alpine-client-*.png")
	if err != nil {
		return nil, err
	}

	data, err := assets.ReadFile(path)
	if err != nil {
		return nil, err
	}

	_, err = tempFile.Write(data)
	if err != nil {
		return nil, err
	}
	err = tempFile.Close()
	if err != nil {
		return nil, err
	}

	return tempFile, nil
}
