package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
)

func SystemInformation() (OperatingSystem, Architecture) {
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

func (sys OperatingSystem) JavaExecutable() string {
	if sys == Windows {
		return "javaw.exe"
	}
	return "java"
}

func GetFromURL(ctx context.Context, url string) (io.ReadCloser, error) {
	AddBreadcrumb(ctx, "making request to "+url)
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

func DownloadFromURL(ctx context.Context, url string, path string) error {
	body, err := GetFromURL(ctx, url)
	if err != nil {
		return err
	}
	defer func() {
		if err = body.Close(); err != nil {
			CaptureErrExit(ctx, err)
		}
	}()

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if err = file.Close(); err != nil {
			CaptureErrExit(ctx, err)
		}
	}()

	_, err = io.Copy(file, body)
	return err
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
