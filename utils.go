package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

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

var httpClient = &http.Client{
	Transport: &http.Transport{
		DisableKeepAlives:     true,
		ResponseHeaderTimeout: 15 * time.Second,
	},
}

func getFromURL(ctx context.Context, url string) (io.ReadCloser, error) {
	const maxAttempts = 4
	var statusCode int

	for i := range maxAttempts {
		if i > 0 {
			<-time.After(time.Second * time.Duration(2<<i)) // Exponential backoff
		}
		sentry.Breadcrumb(ctx, fmt.Sprintf("[%d] making request to %s", i+1, url))

		request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("User-Agent", fmt.Sprintf("Pinnacle/%s (%s; %s)", version, Sys, Arch))

		response, err := httpClient.Do(request)
		if err != nil {
			sentry.Breadcrumb(ctx, fmt.Sprintf("[%d] request error: %v", i+1, err), sentry.LevelError)
			continue
		}

		statusCode = response.StatusCode
		sentry.Breadcrumb(ctx, fmt.Sprintf("[%d] status code: %d", i+1, statusCode))
		if statusCode == http.StatusOK {
			return response.Body, nil
		}

		err = response.Body.Close()
		if err != nil {
			sentry.Breadcrumb(ctx, fmt.Sprintf("[%d] failed to close body: %v", i+1, err), sentry.LevelError)
		}
	}
	return nil, errors.New("internet failure")
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
