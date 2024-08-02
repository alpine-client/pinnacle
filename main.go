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

	type task struct {
		job func(context.Context) error
		f   func(context.Context, error) error
	}

	go func() {
		for _, t := range []task{
			{
				job: setup,
				f:   cleanup,
			},
			{
				job: checkJava,
				f:   download,
			},
			{
				job: checkLauncher,
				f:   download,
			},
			{
				job: startLauncher,
				f:   cleanup,
			},
		} {
			err := t.f(ctx, t.job(ctx))
			if err != nil {
				_ = cleanup(ctx, err)
				break
			}
		}

		ui.Close()
		done <- true
	}()

	<-done
	if logFile != nil {
		sentry.CaptureErr(ctx, logFile.Close())
	}
}
