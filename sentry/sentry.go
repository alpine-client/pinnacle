package sentry

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/ncruces/zenity"
)

// sentryDSN is set via go build -ldflags "-X main.sentryDSN=our_dsn".
var sentryDSN string

const (
	LevelWarning = sentry.LevelWarning
	LevelError   = sentry.LevelError
)

func Start(release string) {
	if sentryDSN != "" {
		_ = sentry.Init(sentry.ClientOptions{
			Dsn:     sentryDSN,
			Release: "pinnacle@" + release,
		})
	}
}

func Flush(timeout time.Duration) {
	sentry.Flush(timeout)
}

func Recover(ctx context.Context) {
	if r := recover(); r != nil {
		Breadcrumb(ctx, fmt.Sprintf("recovered panic: %v", r), LevelWarning)
	}
}

type contextKey string

func NewContext(task string) context.Context {
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
	var lvl sentry.Level
	if len(level) == 0 {
		lvl = sentry.LevelInfo
	} else {
		lvl = level[0]
	}
	hub := sentry.GetHubFromContext(ctx)
	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Category: ctx.Value(contextKey("task")).(string),
		Message:  desc,
		Level:    lvl,
	}, nil)
}

// CaptureErr reports an error to Sentry but does not exit the program.
func CaptureErr(ctx context.Context, err error) *sentry.EventID {
	if err == nil {
		return nil
	}
	Breadcrumb(ctx, err.Error(), sentry.LevelError)
	return sentry.GetHubFromContext(ctx).CaptureException(err)
}

// CaptureErrExit sends the error to sentry and displays a pop-up for the user
// Ensures that only the first pop-up displays in the event of multiple errors.
// Also adds a breadcrumb to the provided hub.
func CaptureErrExit(ctx context.Context, err error) {
	if err == nil {
		return
	}

	id := CaptureErr(ctx, err)
	choice := zenity.Error(
		err.Error()+"\n\nCode: "+string(*id),
		zenity.Title("Error"),
		zenity.OKLabel("Close"),
		zenity.ExtraButton("Help"),
		zenity.ErrorIcon,
	)

	if errors.Is(choice, zenity.ErrExtraButton) {
		openSupportWebsite()
	}

	os.Exit(1)
}

const SupportURL string = "https://discord.alpineclient.com"

// openSupportWebsite tries to open the specified URL in the default browser.
func openSupportWebsite() {
	var err error

	switch runtime.GOOS {
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", SupportURL).Run()
	case "linux":
		err = exec.Command("xdg-open", SupportURL).Run()
	case "darwin":
		err = exec.Command("open", SupportURL).Run()
	}

	if err != nil {
		// None of the above worked. Create new popup with url.
		_ = zenity.Info(
			"Please visit "+SupportURL+" for assistance.",
			zenity.Title("Error"),
			zenity.InfoIcon,
		)
	}
}
