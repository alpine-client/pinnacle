package sentry

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/getsentry/sentry-go"
)

var enabled bool

func Start(release string, dsn string) error {
	if dsn == "" {
		return errors.New("missing sentry DSN")
	}
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		AttachStacktrace: true,
		Release:          "pinnacle@" + release,
		Transport:        sentry.NewHTTPSyncTransport(),
	})
	if err != nil {
		return err
	}
	enabled = true
	return nil
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

func Breadcrumb(ctx context.Context, desc string, level slog.Level) {
	if !enabled {
		return
	}

	var lvl sentry.Level
	switch level {
	case slog.LevelDebug:
		lvl = sentry.LevelDebug
	case slog.LevelInfo:
		lvl = sentry.LevelInfo
	case slog.LevelWarn:
		lvl = sentry.LevelWarning
	case slog.LevelError:
		lvl = sentry.LevelError
	}

	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		hub.AddBreadcrumb(&sentry.Breadcrumb{
			Category: ctx.Value(contextKey("task")).(string),
			Message:  desc,
			Level:    lvl,
		}, nil)
	}
}

// CaptureErr reports an error to Sentry.
func CaptureErr(ctx context.Context, err error) *sentry.EventID {
	if err == nil || !enabled {
		return nil
	}
	Breadcrumb(ctx, err.Error(), slog.LevelError)
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		return hub.CaptureException(err)
	}
	return nil
}
