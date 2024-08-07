package ui

import (
	"context"
	"errors"
	"log"
	"os/exec"
	"runtime"
	"time"

	"github.com/alpine-client/pinnacle/sentry"
	"github.com/ncruces/zenity"
)

// DisplayError closes the progress-bar, sends the error to sentry and displays a pop-up for the user
// Also adds a breadcrumb to the provided sentry hub connected to the context.
func DisplayError(ctx context.Context, err error) {
	if err == nil {
		return
	}

	Close() // close progress bar

	message := err.Error()

	id := sentry.CaptureErr(ctx, err)
	if id != nil {
		message += "\n\nCode: " + string(*id)
	}

	sentry.Flush(2 * time.Second)

	choice := zenity.Error(
		message+"\n\nJoin our Discord for help.",
		zenity.Title("Error"),
		zenity.OKLabel("Close"),
		zenity.ExtraButton("Help (Discord)"),
		zenity.ErrorIcon,
	)

	if errors.Is(choice, zenity.ErrExtraButton) {
		openSupportWebsite()
	}
}

// openSupportWebsite tries to open the specified URL in the default browser.
func openSupportWebsite() {
	const supportURL string = "https://discord.alpineclient.com"
	var err error

	switch runtime.GOOS {
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", supportURL).Run()
	case "linux":
		err = exec.Command("xdg-open", supportURL).Run()
	case "darwin":
		err = exec.Command("open", supportURL).Run()
	}

	if err != nil {
		log.Printf("[ERROR] %v", err)
		// None of the above worked. Create new popup with url.
		err = zenity.Info(
			"Please visit "+supportURL+" for assistance.",
			zenity.Title("Error"),
			zenity.InfoIcon,
		)
		if err != nil {
			log.Printf("[ERROR] %v", err)
		}
	}
}
