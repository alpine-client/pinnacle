package ui

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/alpine-client/pinnacle/sentry"
	"github.com/ncruces/zenity"
)

// DisplayError closes the progress-bar, sends the error to sentry and displays a pop-up for the user
// Also adds a breadcrumb to the provided sentry hub connected to the context.
func DisplayError(ctx context.Context, err error, logFile *os.File) error {
	if err == nil {
		return nil
	}

	Close() // close progress bar

	message := err.Error()

	var logContent string
	if logFile != nil {
		logFile.Close()
		log.Println("reading log file again at: " + logFile.Name())
		logFileToRead, lerr := os.Open(logFile.Name())
		if lerr != nil {
			log.Printf("error opening log file %q: %v", logFile.Name(), lerr)
		} else {
			defer logFileToRead.Close()
			logData, rerr := io.ReadAll(logFileToRead)
			if rerr != nil {
				log.Println(errors.Unwrap(rerr))
			} else {
				log.Println("attaching log file to sentry report")
				logContent = string(logData)
				log.Printf("length of log: %d\n", len(logContent))
			}
		}

	}

	id := sentry.CaptureErr(ctx, err, logContent)
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
		return openSupportWebsite()
	}

	return nil
}

// openSupportWebsite tries to open the specified URL in the default browser.
func openSupportWebsite() error {
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
		// None of the above worked. Create new popup with url.
		_ = zenity.Info(
			"Please visit "+supportURL+" for assistance.",
			zenity.Title("Error"),
			zenity.InfoIcon,
		)
		return err
	}
	return nil
}
