package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/ncruces/zenity"
)

func isBadRecordMacErr(err error) bool {
	// Since errors can be wrapped, we need to unwrap it first
	unwrappedErr := errors.Unwrap(err)
	if unwrappedErr == nil {
		unwrappedErr = err // There was no wrapped error, so we use the original
	}

	// Check if the error message contains the specific TLS bad record MAC message
	return strings.Contains(unwrappedErr.Error(), "tls: bad record MAC")
}

// CaptureAndExit sends the error to sentry and displays a pop-up for the user
// Ensures that only the first pop-up displays in the event of multiple errors.
func CaptureAndExit(err error, hub *sentry.Hub) {
	if err == nil {
		return
	}

	// Send error to sentry
	eventID := hub.CaptureException(err)
	errID := *eventID
	message := err.Error()

	// Override message in known cases
	if isBadRecordMacErr(err) {
		message += "\n\nPlease make sure your system clock is set correctly."
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		message += "\n\nPlease make sure Alpine Client is not already running."
	}

	// Display popup
	choice := zenity.Error(
		fmt.Sprintf("%s\n\nCode: %s", message, errID),
		zenity.Title("Error"),
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

// openSupportWebsite tries to open the specified URL in the default browser.
func openSupportWebsite() {
	var err error
	switch Sys {
	case Windows:
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", SupportURL).Run()
	case Mac:
		err = exec.Command("open", SupportURL).Run()
	case Linux:
		err = exec.Command("xdg-open", SupportURL).Run()
	default:
		err = errors.New("unable to open support page")
	}
	if err != nil {
		// None of the above worked. Create new popup with url.
		_ = zenity.Info(
			fmt.Sprintf("Please visit %s for assistance.", SupportURL),
			zenity.Title("Error"), zenity.InfoIcon,
		)
	}
}
