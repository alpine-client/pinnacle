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

	cimgui "github.com/AllenDang/cimgui-go"
	"github.com/AllenDang/giu"
	"github.com/alpine-client/pinnacle/sentry"
	"github.com/ncruces/zenity"
)

const (
	LogoSize     float32 = 80
	WindowWidth  int     = 377
	WindowHeight int     = 144
)

var (
	logoI  *os.File
	window *giu.MasterWindow
	dialog zenity.ProgressDialog
	tasks  []*ProgressiveTask
)

func Render() {
	if window != nil {
		window.Run(defaultUI)
	} else {
		runZenity()
	}
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

	ws := cimgui.CurrentPlatformIO().Monitors().Data.WorkSize()
	window.SetPos(int((ws.X-float32(WindowWidth))/2), int((ws.Y-float32(WindowHeight))/2))

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

func defaultUI() {
	SetupStyle()
	defer PopStyle()

	w := []giu.Widget{
		giu.Dummy(0, scaleDivider(2)),
		giu.ImageWithFile(logoI.Name()).Size(LogoSize, LogoSize),
	}

	var most *ProgressiveTask
	for _, task := range tasks {
		if task.progress < 1 {
			if most == nil || task.progress > most.progress {
				most = task
			}
		}
	}
	if most != nil {
		w = append(w, giu.Label(most.label))
		w = append(w, giu.ProgressBar(most.progress).Size(scaleValueX(WindowWidth)*0.75, scaleValueY(3)))
		w = append(w, giu.Dummy(0, scaleDivider(2)))
	}
	giu.SingleWindow().Layout(giu.Align(giu.AlignCenter).To(w...))
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

func scaleDivider(value float32) float32 {
	_, yScale := giu.Context.Backend().ContentScale()
	if yScale > 1.0 {
		value *= 2
	}
	return value * yScale
}

func scaleValueX(value int) float32 {
	xScale, _ := giu.Context.Backend().ContentScale()
	return float32(value) * xScale
}

func scaleValueY(value int) float32 {
	_, yScale := giu.Context.Backend().ContentScale()
	return float32(value) * yScale
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
	temp, err := os.CreateTemp("", "alpine-client-*.png")
	if err != nil {
		return nil, err
	}

	data, err := assets.ReadFile(path)
	if err != nil {
		return nil, err
	}

	_, err = temp.Write(data)
	if err != nil {
		return nil, err
	}

	err = temp.Close()
	if err != nil {
		return nil, err
	}

	return temp, nil
}
