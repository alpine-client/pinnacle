package main

import (
	"context"
	"time"

	"github.com/alpine-client/pinnacle/sentry"
	"github.com/alpine-client/pinnacle/ui"
)

var (
	Sys     OperatingSystem
	Arch    Architecture
	version string
)

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

	done := make(chan bool)
	defer close(done)

	go func() {
		tasks := []func(c context.Context) error{
			setup,
			checkJRE,
			checkLauncher,
			startLauncher,
		}
		for _, task := range tasks {
			err := task(ctx)
			if err != nil {
				cleanup()
				ui.Close()
				ui.DisplayError(ctx, err)
				break
			}
		}
		time.Sleep(20 * time.Minute)
		ui.Close()
		done <- true
	}()

	ui.Render()

	<-done
	if logFile != nil {
		sentry.CaptureErr(ctx, logFile.Close())
	}
}
