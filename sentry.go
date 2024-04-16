package main

import (
	"context"
	"os"
	"runtime"

	"github.com/getsentry/sentry-go"
)

// sentryDSN is set via go build -ldflags "-X main.sentryDSN=our_dsn".
var sentryDSN string

func StartSentry(release string) {
	if sentryDSN != "" {
		_ = sentry.Init(sentry.ClientOptions{
			Dsn:     sentryDSN,
			Release: "pinnacle@" + release,
		})
	}
}

type ContextKey string

func CreateSentryCtx(task string) context.Context {
	name, _ := os.Hostname()
	localHub := sentry.CurrentHub().Clone()
	localHub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("Task", task)
		scope.SetTag("OS", runtime.GOOS)
		scope.SetTag("Arch", runtime.GOARCH)
		scope.SetUser(sentry.User{Name: name})
		scope.SetLevel(sentry.LevelInfo)
	})
	ctx := context.WithValue(context.Background(), ContextKey("task"), task)
	return sentry.SetHubOnContext(ctx, localHub)
}
