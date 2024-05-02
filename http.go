package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alpine-client/pinnacle/sentry"
	"github.com/alpine-client/pinnacle/ui"
)

var httpClient = &http.Client{
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
	},
}

func getFromURL(ctx context.Context, url string) (io.ReadCloser, error) {
	const maxAttempts = 4
	var statusCode int

	for i := range maxAttempts {
		ui.UpdateProgress(1)
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

func isUpdateAvailable(c context.Context) bool {
	req, err := http.NewRequestWithContext(c, http.MethodGet, GitHubReleaseURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", fmt.Sprintf("alpine-client/pinnacle/%s (%s)", version, SupportEmail))

	resp, err := httpClient.Do(req)
	if err != nil {
		return false
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Printf("unable to close response body: %v", err)
		}
	}()

	type details struct {
		TagName    string `json:"tag_name"`
		PreRelease bool   `json:"prerelease"`
	}
	var result details
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return false
	}

	if len(strings.Split(version, ".")) != 3 {
		return false
	}

	if !result.PreRelease && version != result.TagName {
		return true
	}

	return false
}
