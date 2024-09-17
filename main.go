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
	if err := setup(); err != nil {
		panic(err)
	}

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
		c   context.Context
		job func(context.Context) error
		f   func(context.Context, error) error
	}

	go func() {
		for _, t := range []task{
			{
				c:   sentry.NewContext(ctx, "java"),
				job: checkJava,
				f:   download,
			},
			{
				c:   sentry.NewContext(ctx, "launcher"),
				job: checkLauncher,
				f:   download,
			},
			{
				c:   sentry.NewContext(ctx, "start"),
				job: startLauncher,
				f:   cleanup,
			},
		} {
			err := t.f(t.c, t.job(t.c))
			if err != nil {
				_ = cleanup(t.c, err)
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
