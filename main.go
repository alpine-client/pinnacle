package main

import (
	"context"
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
var sentryDSN string

func main() {
	sentry.Start(version, sentryDSN)
	defer sentry.Flush(2 * time.Second)

	Sys, Arch = systemInformation()

	Run()
}

func Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if isUpdateAvailable(ctx) {
		ui.NotifyNewUpdate()
	}

	ui.Setup(ctx, assets)

	err := os.MkdirAll(alpinePath(), os.ModePerm)
	if err != nil {
		ui.DisplayError(ctx, err)
		return
	}

	done := make(chan bool)
	go runTasks(done)

	ui.Render()

	<-done
	close(done)
}
