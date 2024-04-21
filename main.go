package main

import (
	"embed"
	"os"
	"time"

	"github.com/alpine-client/pinnacle/sentry"
	"github.com/alpine-client/pinnacle/ui"
)

var (
	Sys     OperatingSystem
	Arch    Architecture
	version string
)

//go:embed assets/*
var assets embed.FS

func main() {
	sentry.Start(version)
	defer sentry.Flush(2 * time.Second)

	Sys, Arch = systemInformation()

	Run()
}

func Run() {
	ctx := sentry.NewContext("Run")

	err := os.MkdirAll(alpinePath(), os.ModePerm)
	sentry.CaptureErrExit(ctx, err)

	window := ui.NewWindow(ctx, assets) // nil if panics

	done := make(chan bool)
	go runTasks(window, done)

	if window != nil {
		window.Run(ui.RenderDefault)
	} else {
		dialog := ui.NewZenityWindow(ctx)
		<-done

		_ = dialog.Complete()
	}
}
