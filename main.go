package main

import (
	"context"
	"runtime"
	"time"

	"github.com/alpine-client/pinnacle/sentry"
	"github.com/alpine-client/pinnacle/ui"
)

var version string

func main() {
	p := &Pinnacle{
		os:      OperatingSystem(runtime.GOOS),
		arch:    Architecture(runtime.GOARCH),
		version: version,
	}

	if err := p.setup(); err != nil {
		panic(err)
	}
	defer sentry.Flush(2 * time.Second)

	p.Run()
}

func (p *Pinnacle) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if p.isUpdateAvailable(ctx) {
		ui.NotifyNewUpdate()
	}

	done := make(chan bool)
	defer close(done)

	type task struct {
		c context.Context
		j func(context.Context) error
		f func(context.Context, error) error
	}

	go func() {
		for _, t := range []task{
			{
				c: sentry.NewContext(ctx, "java"),
				j: p.checkJava,
				f: p.download,
			},
			{
				c: sentry.NewContext(ctx, "launcher"),
				j: p.checkLauncher,
				f: p.download,
			},
			{
				c: sentry.NewContext(ctx, "start"),
				j: p.startLauncher,
				f: p.error,
			},
		} {
			err := t.f(t.c, t.j(t.c))
			if err != nil {
				p.cleanup(t.c, err)
				break
			}
		}

		ui.Close()
		done <- true
	}()

	<-done
	if logFile != nil {
		p.CaptureErr(ctx, logFile.Close())
	}
}
