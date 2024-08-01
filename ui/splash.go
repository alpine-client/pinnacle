package ui

import (
	"embed"
	"image/color"
	"log"
	"sync"
	"time"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	gs "github.com/gesellix/gioui-splash"
)

var (
	splashWidget *gs.Splash
	splashWindow app.Window
)

func SetupSplash(assets embed.FS) {
	logo, err := LoadImage(assets, "assets/logo.png")
	if err != nil {
		log.Fatal(err)
	}
	size := logo.Bounds().Size()
	sizeXDp := unit.Dp(size.X)
	sizeYDp := unit.Dp(size.Y)

	options := []app.Option{
		app.Title("Alpine Client"),
		app.Size(sizeXDp, sizeYDp),
		app.MinSize(sizeXDp, sizeYDp),
		app.MaxSize(sizeXDp, sizeYDp),
		app.Decorated(true),
	}

	splashWindow = app.Window{}
	splashWindow.Option(options...)
	splashWindow.Perform(system.ActionCenter)

	splashWidget = gs.NewSplash(
		logo,
		// (Bottom-Top) sets the height of the progress bar
		layout.Inset{
			Top:    5,
			Bottom: 10,
			Left:   10,
			Right:  10,
		},
		color.NRGBA{R: 166, G: 38, B: 57, A: 127},
	)

	go func() {
		tick := time.NewTicker(50 * time.Millisecond)
		defer tick.Stop()

		for {
			select {
			case <-tick.C:
				var most *ProgressiveTask
				for _, task := range tasks {
					if task.progress < 1 {
						if most == nil || task.progress > most.progress {
							most = task
						}
					}
				}
				if most != nil {
					if most.progress < 1 {
						splashWidget.SetProgress(float64(most.progress))
					} else {
						splashWidget.SetProgress(0)
					}
				}
				// The widget will not be updated until the next FrameEvent.
				// We're going to trigger that event now, so that
				// the changed progress will be visible.
				splashWindow.Invalidate()
			}
		}
	}()
}

func RunSplash() {
	// TODO work around https://todo.sr.ht/~eliasnaur/gio/602
	// this should only be required shortly after creating the window w.
	performCenter := sync.OnceFunc(func() {
		splashWindow.Perform(system.ActionCenter)
	})
	var ops op.Ops
	for {
		switch e := splashWindow.Event().(type) {
		case app.FrameEvent:
			// TODO work around https://todo.sr.ht/~eliasnaur/gio/602
			// this should only be required shortly after creating the window w.
			performCenter()
			gtx := app.NewContext(&ops, e)
			splashWidget.Layout(gtx)
			e.Frame(gtx.Ops)
		case app.DestroyEvent:
			// Omitted
		}
	}
}

func CloseSplash() {
	splashWindow.Perform(system.ActionClose)
}
