package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"
)

func SystemInformation() (OperatingSystem, Architecture) {
	var sys OperatingSystem
	var arch Architecture
	var err error

	switch runtime.GOOS {
	case "windows":
		sys = Windows
	case "linux":
		sys = Linux
	case "darwin":
		sys = Mac
	default:
		err = errors.New("unsupported operating system")
	}

	switch runtime.GOARCH {
	case "amd64":
		arch = x86
	case "arm64":
		arch = Arm64
	default:
		err = errors.New("unsupported system architecture")
	}

	if err != nil {
		hub := CreateSentryHub("SystemInformation")
		HandleFatalError("Unsupported platform", err, hub) // this call exists program
	}
	return sys, arch
}

func (sys OperatingSystem) JavaExecutable() string {
	if sys == Windows {
		return "javaw.exe"
	}
	return "java"
}

func GetFromUrl(url string) (io.ReadCloser, error) {
	// Create the HTTP client
	client := http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 10 * time.Second,
			}).DialContext,
		},
	}

	// Create the HTTP request
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", fmt.Sprintf("Pinnacle/%s (%s; %s)", Version, Sys, Arch))

	// Perform the HTTP request
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	// Check if request was successful
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received status code: %d", response.StatusCode)
	}

	return response.Body, nil
}

func DownloadFromUrl(url string, path string) error {
	// Create or truncate the file
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Perform the HTTP request
	body, err := GetFromUrl(url)
	if err != nil {
		return err
	}
	defer body.Close()

	// Copy response body to file
	_, err = io.Copy(file, body)
	return err
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
