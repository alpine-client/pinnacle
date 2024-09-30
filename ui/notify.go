package ui

import (
	"log"

	"github.com/ncruces/zenity"
)

const (
	downloadURL = "https://alpineclient.com/download"
)

func NotifyNewUpdate() {
	const msg = "Update available!\n\nPlease visit " + downloadURL

	log.Println(msg)
	_ = zenity.Notify(msg, zenity.Title(WindowTitle))
}
