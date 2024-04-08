package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/AllenDang/giu"
	"github.com/getsentry/sentry-go"
)

var (
	Sys        OperatingSystem
	Arch       Architecture
	WorkingDir string
	version    string
	logo       *giu.Texture
)

func main() {

	StartSentry(version)
	hub := CreateSentryHub("main")
	defer sentry.Flush(2 * time.Second)

	Sys, Arch = SystemInformation()
	WorkingDir = getAlpinePath()

	err := os.MkdirAll(WorkingDir, os.ModePerm)
	CaptureAndExit(err, hub)

	window := giu.NewMasterWindow(
		"Alpine Client Updater",
		WindowWidth, WindowHeight,
		giu.MasterWindowFlagsFrameless|giu.MasterWindowFlagsNotResizable|giu.MasterWindowFlagsTransparent,
	)
	window.SetBgColor(color.Transparent)

	runTasks(window)

	// Load textures
	img, err := loadImage(IconBytes)
	CaptureAndExit(err, hub)
	window.SetIcon([]image.Image{img})

	img, err = loadImage(LogoBytes)
	CaptureAndExit(err, hub)
	giu.NewTextureFromRgba(img, func(tex *giu.Texture) {
		logo = tex
	})

	// Run main UI loop
	window.Run(drawUI)
}

func runTasks(window *giu.MasterWindow) {
	var wg sync.WaitGroup
	wg.Add(2)

	go BeginJre(&wg)
	go BeginLauncher(&wg)

	go func() {
		wg.Wait()

		hub := CreateSentryHub("runTasks") // wasn't sure what to name it
		jarPath := filepath.Join(WorkingDir, "launcher.jar")
		jrePath := filepath.Join(WorkingDir, "jre", "17", "extracted", "bin", Sys.JavaExecutable())

		args := []string{jrePath}

		if Sys == Mac {
			args = append(args, "-XstartOnFirstThread")
		}

		args = append(
			args,
			"-Xms256M",
			"-Xmx1G",
			"-jar",
			jarPath,
		)

		if version != "" {
			args = append(args, "--pinnacle-version", version)
		}

		processAttr := &os.ProcAttr{
			Dir:   WorkingDir,
			Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		}

		proc, err := os.StartProcess(jrePath, args, processAttr)
		CaptureAndExit(err, hub)

		err = proc.Release()
		CaptureAndExit(err, hub)

		window.SetShouldClose(true)
	}()
}

func drawUI() {
	SetupStyle()
	giu.SingleWindow().Layout(
		giu.Align(giu.AlignCenter).To(
			giu.Dummy(0, scaleDivider(6)),
			giu.Image(logo).Size(LogoSize, LogoSize),
			giu.Dummy(0, scaleDivider(6)),
			giu.ProgressBar(float32(CompletedTasks)/float32(TotalTasks)).Size(scaleValue(WindowWidth)*0.75, scaleValue(5)),
		),
	)
	PopStyle()
}

func loadImage(data []uint8) (image.Image, error) {
	img, err := png.Decode(bytes.NewReader(data))
	return img, err
}

func scaleDivider(value float32) float32 {
	scale := giu.Context.GetPlatform().GetContentScale()
	if scale > 1.0 {
		value = value * 2
	}
	return value * scale
}

func scaleValue(value int) float32 {
	scale := giu.Context.GetPlatform().GetContentScale()
	return float32(value) * scale
}

// GetAlpinePath returns the absolute path of Alpine Client's
// data directory based on the user's operating system.
//
// Windows - %AppData%\.alpineclient
// Mac - $HOME/Library/Application Support/alpineclient
// Linux - $HOME/.alpineclient
//
// note: The missing '.' for macOS is intentional.
func getAlpinePath() string {
	var baseDir string
	var dir string

	switch Sys {
	case Windows:
		baseDir = os.Getenv("AppData")
		dir = filepath.Join(baseDir, ".alpineclient")
	case Mac:
		baseDir = os.Getenv("HOME")
		dir = filepath.Join(baseDir, "Library", "Application Support", "alpineclient")
	case Linux:
		baseDir = os.Getenv("HOME")
		dir = filepath.Join(baseDir, ".alpineclient")
	}
	return dir
}
