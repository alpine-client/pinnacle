package ui

import (
	"log/slog"

	"github.com/ncruces/zenity"
)

const (
	downloadURL = "https://alpineclient.com/download"
)

func NotifyNewUpdate(l *slog.Logger) {
	const msg = "Update available!\n\nPlease visit " + downloadURL

	l.Info(msg)
	_ = zenity.Notify(msg, zenity.Title(WindowTitle))
}
