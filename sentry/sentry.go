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

type Client struct {
	logger  *slog.Logger
	enabled bool
}

func New(logger *slog.Logger) *Client {
	return &Client{
		logger: logger,
	}
}

func (c *Client) Start(release string, dsn string) error {
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
	c.enabled = true
	return nil
}

func Flush(timeout time.Duration) {
	sentry.Flush(timeout)
}

type contextKey string

func (c *Client) NewContext(parent context.Context, task string) context.Context {
	if !c.enabled {
		return parent
	}
	name, err := os.Hostname()
	if err != nil {
		name = "unknown"
		c.logger.WarnContext(parent, err.Error())
	}
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

func (c *Client) Breadcrumb(ctx context.Context, desc string, level slog.Level) {
	if !c.enabled {
		c.logger.ErrorContext(ctx, desc)
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
		if task, ok := ctx.Value(contextKey("task")).(string); ok {
			hub.AddBreadcrumb(&sentry.Breadcrumb{
				Category: task,
				Message:  desc,
				Level:    lvl,
			}, nil)
		}
	}
}

// CaptureErr reports an error to Sentry.
func (c *Client) CaptureErr(ctx context.Context, err error, attachment ...string) *sentry.EventID {
	if err == nil {
		return nil
	}
	if !c.enabled {
		c.logger.ErrorContext(ctx, err.Error())
		return nil
	}
	c.Breadcrumb(ctx, err.Error(), slog.LevelError)
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		if len(attachment) > 0 {
			hub.ConfigureScope(func(scope *sentry.Scope) {
				scope.AddAttachment(&sentry.Attachment{
					Filename: "updater.log",
					Payload:  []byte(attachment[0]),
				})
			})
		}
		return hub.CaptureException(err)
	}
	return nil
}
