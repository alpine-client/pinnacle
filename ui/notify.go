package ui

import (
	"github.com/ncruces/zenity"
)

const downloadURL = "https://alpineclient.com/download"

func NotifyNewUpdate() {
	_ = zenity.Notify("New version available!\n\nPlease visit " + downloadURL)
}
