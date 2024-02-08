package main

import (
	"github.com/ncruces/zenity"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
)

var (
	Version    string
	Sys        OperatingSystem
	Arch       Architecture
	WorkingDir string
)

func main() {

	StartSentry(Version)
	hub := CreateSentryHub("main")
	defer sentry.Flush(2 * time.Second)

	Sys, Arch = SystemInformation()
	WorkingDir = getAlpinePath()

	err := os.MkdirAll(WorkingDir, os.ModePerm)
	HandleFatalError("Failed to create working directory", err, hub)

	dlg, _ := zenity.Progress(
		zenity.Title("Updating Alpine Client"),
		zenity.Pulsate(),
		zenity.NoCancel(),
		zenity.AutoClose(),
	)

	// Channel to signal when runTasks is done
	done := make(chan bool)
	go runTasks(done)
	<-done // Wait for runTasks to signal completion

	dlg.Complete()
}

func runTasks(done chan bool) {
	var wg sync.WaitGroup
	wg.Add(2)

	go BeginJre(&wg)
	go BeginLauncher(&wg)

	go func() {
		wg.Wait()

		hub := CreateSentryHub("runTasks") // wasn't sure what to name it
		jarPath := filepath.Join(WorkingDir, "launcher.jar")
		jrePath := filepath.Join(WorkingDir, "jre", "17", "extracted", "bin", Sys.JavaExecutable())

		args := []string{jrePath}

		if Sys == Mac {
			args = append(args, "-XstartOnFirstThread")
		}

		args = append(
			args,
			"-Xms256M",
			"-Xmx1G",
			"-jar",
			jarPath,
		)

		if Version != "" {
			args = append(args, "--pinnacle-version", Version)
		}

		processAttr := &os.ProcAttr{
			Dir:   WorkingDir,
			Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		}

		proc, err := os.StartProcess(jrePath, args, processAttr)
		HandleFatalError("Failed to start launcher process", err, hub)

		err = proc.Release()
		HandleFatalError("Failed to detach launcher process", err, hub)
		done <- true //  Signal that runTasks is complete
	}()
}

// GetAlpinePath returns the absolute path of Alpine Client's
// data directory based on the user's operating system.
//
// Windows - %AppData%\.alpineclient
// Mac - $HOME/Library/Application Support/alpineclient
// Linux - $HOME/.alpineclient
//
// note: The missing '.' for macOS is intentional.
func getAlpinePath() string {
	var baseDir string
	var dir string

	switch Sys {
	case Windows:
		baseDir = os.Getenv("AppData")
		dir = filepath.Join(baseDir, ".alpineclient")
	case Mac:
		baseDir = os.Getenv("HOME")
		dir = filepath.Join(baseDir, "Library", "Application Support", "alpineclient")
	case Linux:
		baseDir = os.Getenv("HOME")
		dir = filepath.Join(baseDir, ".alpineclient")
	}
	return dir
}
