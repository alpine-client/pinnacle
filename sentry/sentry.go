package sentry

import (
	"context"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/getsentry/sentry-go"
)

var enabled bool

const (
	LevelWarning = sentry.LevelWarning
	LevelError   = sentry.LevelError
)

func Start(release string, dsn string) {
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:       dsn,
		Release:   "pinnacle@" + release,
		Transport: sentry.NewHTTPSyncTransport(),
	}); err != nil {
		log.Printf("unable to start sentry: %v", err)
		// TODO: log to file
	} else {
		enabled = true
	}
}

func Flush(timeout time.Duration) {
	sentry.Flush(timeout)
}

type contextKey string

func NewContext(task string) context.Context {
	if !enabled {
		return context.Background()
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
	ctx := context.WithValue(context.Background(), contextKey("task"), task)
	return sentry.SetHubOnContext(ctx, localHub)
}

func Breadcrumb(ctx context.Context, desc string, level ...sentry.Level) {
	if !enabled {
		return
	}
	var lvl sentry.Level
	if len(level) == 0 {
		lvl = sentry.LevelInfo
	} else {
		lvl = level[0]
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
	if err == nil || !enabled {
		return nil
	}
	Breadcrumb(ctx, err.Error(), sentry.LevelError)
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		return hub.CaptureException(err)
	}
	return nil
}
