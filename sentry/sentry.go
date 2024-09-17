package sentry

import (
	"context"
	"log"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/getsentry/sentry-go"
)

var enabled bool

const (
	LevelError = sentry.LevelError
)

func Start(release string, dsn string) {
	if dsn == "" {
		slog.Warn("missing sentry DSN")
		return
	}
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		AttachStacktrace: true,
		Release:          "pinnacle@" + release,
		Transport:        sentry.NewHTTPSyncTransport(),
	})
	if err != nil {
		slog.Warn("unable to start sentry: %v", err)
		return
	}
	enabled = true
}

func Flush(timeout time.Duration) {
	sentry.Flush(timeout)
}

type contextKey string

func NewContext(parent context.Context, task string) context.Context {
	if !enabled {
		return parent
	}
	name, _ := os.Hostname()
	localHub := sentry.CurrentHub().Clone()
	localHub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("Task", task)
		scope.SetTag("OS", runtime.GOOS)
		scope.SetTag("Arch", runtime.GOARCH)
		scope.SetUser(sentry.User{Name: name})
		scope.SetLevel(sentry.LevelInfo)
	})
	ctx := context.WithValue(parent, contextKey("task"), task)
	return sentry.SetHubOnContext(ctx, localHub)
}

func Breadcrumb(ctx context.Context, desc string, level ...sentry.Level) {
	var lvl sentry.Level
	if len(level) == 0 {
		lvl = sentry.LevelInfo
	} else {
		lvl = level[0]
	}
	log.Printf("%s %s", lvl, desc)
	if !enabled {
		return
	}
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		hub.AddBreadcrumb(&sentry.Breadcrumb{
			Category: ctx.Value(contextKey("task")).(string),
			Message:  desc,
			Level:    lvl,
		}, nil)
	}
}

// CaptureErr reports an error to Sentry but does not exit the program.
func CaptureErr(ctx context.Context, err error) *sentry.EventID {
	if err == nil {
		return nil
	}
	slog.Error(err.Error())
	if !enabled {
		return nil
	}
	Breadcrumb(ctx, err.Error(), sentry.LevelError)
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		return hub.CaptureException(err)
	}
	return nil
}
