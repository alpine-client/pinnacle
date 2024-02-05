package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/getsentry/sentry-go"
	"github.com/ncruces/zenity"
)

const supportURL = "https://discord.alpineclient.com"

// HandleFatalError sends the error to sentry and displays a pop-up for the user
// Ensures that only the first pop-up displays in the event of multiple errors.
func HandleFatalError(message string, err error, hub *sentry.Hub) {
	if err != nil {
		// Send error to sentry
		hub.CaptureException(err)

		// Display popup
		choice := zenity.Error(
			message, zenity.Title("Error"),
			zenity.OKLabel("Close"),
			zenity.ExtraButton("Help"),
			zenity.ErrorIcon,
		)

		if errors.Is(choice, zenity.ErrExtraButton) {
			openSupportWebsite()
		}

		// Exit program
		os.Exit(1)
	}
}

// openURL tries to open the specified URL in the default browser.
func openSupportWebsite() {
	var err error
	switch Sys {
	case Windows:
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", supportURL).Start()
	case Mac:
		err = exec.Command("open", supportURL).Start()
	default:
		// Chances are we are on some form of Linux, this is our best bet
		err = exec.Command("xdg-open", supportURL).Start()
	}
	if err == nil {
		// None of the above worked. Create new popup with url.
		_ = zenity.Info(
			fmt.Sprintf("Please visit %s for assistance.", supportURL),
			zenity.Title("Error"), zenity.InfoIcon,
		)
	}
}
