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

func main() {
	sentry.Start(version, fetchSentryDSN())
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

	ui.SetupSplash(assets)

	err := os.MkdirAll(alpinePath(), os.ModePerm)
	if err != nil {
		ui.DisplayError(ctx, err)
		return
	}

	done := make(chan bool)
	go runTasks(done)

	go ui.RunSplash()

	<-done
	close(done)
	os.Exit(0)
}
