package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/alpine-client/pinnacle/sentry"
)

func systemInformation() (OperatingSystem, Architecture) {
	var sys OperatingSystem
	var arch Architecture

	switch runtime.GOOS {
	case "windows":
		sys = Windows
	case "linux":
		sys = Linux
	case "darwin":
		sys = Mac
	default:
		panic("unsupported operating system")
	}

	switch runtime.GOARCH {
	case "amd64":
		arch = x86
	case "arm64":
		arch = Arm64
	default:
		panic("unsupported system architecture")
	}

	return sys, arch
}

func (sys OperatingSystem) javaExecutable() string {
	if sys == Windows {
		return "javaw.exe"
	}
	return "java"
}

// alpinePath returns the absolute path of Alpine Client's
// data directory based on the user's operating system.
//
// Optionally, pass in sub-folder/file names to add
// them to the returned path.
// - Example: alpinePath("jre", "17", "version.json")
//
// Windows - %AppData%\.alpineclient
// Mac - $HOME/Library/Application Support/alpineclient
// Linux - $HOME/.alpineclient
//
// note: The missing '.' for macOS is intentional.
func alpinePath(subs ...string) string {
	var baseDir string
	var dirs []string

	switch Sys {
	case Windows:
		baseDir = os.Getenv("AppData")
		dirs = append(dirs, baseDir, ".alpineclient")
	case Mac:
		baseDir = os.Getenv("HOME")
		dirs = append(dirs, baseDir, "Library", "Application Support", "alpineclient")
	case Linux:
		baseDir = os.Getenv("HOME")
		dirs = append(dirs, baseDir, ".alpineclient")
	}

	return filepath.Join(append(dirs, subs...)...)
}

func getFromURL(ctx context.Context, url string) (io.ReadCloser, error) {
	sentry.Breadcrumb(ctx, "making request to "+url)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", fmt.Sprintf("Pinnacle/%s (%s; %s)", version, Sys, Arch))
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received status code: %d", response.StatusCode)
	}

	return response.Body, nil
}

func downloadFromURL(ctx context.Context, url string, path string) error {
	body, err := getFromURL(ctx, url)
	if err != nil {
		return err
	}
	defer func() {
		sentry.CaptureErr(ctx, body.Close())
	}()

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		sentry.CaptureErr(ctx, file.Close())
	}()

	_, err = io.Copy(file, body)
	return err
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
