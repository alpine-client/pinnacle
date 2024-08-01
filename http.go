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

func getFromURL(ctx context.Context, url string) (*http.Response, error) {
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
			return response, nil
		}

		err = response.Body.Close()
		if err != nil {
			sentry.Breadcrumb(ctx, fmt.Sprintf("[%d] failed to close body: %v", i+1, err), sentry.LevelError)
		}
	}
	return nil, errors.New("internet failure")
}

func downloadFile(ctx context.Context, url string, path string, pt *ui.ProgressiveTask) error {
	resp, err := getFromURL(ctx, url)
	if err != nil {
		return err
	}
	defer func() {
		sentry.CaptureErr(ctx, resp.Body.Close())
	}()

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		sentry.CaptureErr(ctx, file.Close())
	}()

	err = copyResponseWithProgress(file, resp, pt)
	return err
}

func copyResponseWithProgress(dst io.Writer, resp *http.Response, pt *ui.ProgressiveTask) error {
	var written int64
	var err error
	buf := make([]byte, 32*1024)
	src := resp.Body
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = errors.New("invalid write result")
				}
			}
			written += int64(nw)
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
			if pt != nil {
				pt.UpdateProgress(float64(written) / float64(resp.ContentLength))
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return err
}

func fetchSentryDSN() string {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, MetadataURL+"/sentry", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", fmt.Sprintf("alpine-client/pinnacle/%s (%s)", version, SupportEmail))

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("unable to fetch sentry DSN: %v", err)
		return ""
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Printf("unable to close response body: %v", err)
		}
	}()

	type response struct {
		DSN string `json:"dsn"`
	}

	var result response
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		log.Printf("unable to decode sentry DSN response: %v", err)
		return ""
	}

	return result.DSN
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
		TagName     string `json:"tag_name"`
		PreRelease  bool   `json:"prerelease"`
		PublishedAt string `json:"published_at"`
	}

	var result details
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return false
	}

	if result.PreRelease {
		return false
	}

	if version == result.TagName {
		return false
	}

	v := strings.Split(version, ".")
	r := strings.Split(result.TagName, ".")
	if len(v) != 3 || len(r) != 3 {
		return false
	}

	if v[0] == r[0] && v[1] == r[1] {
		// major + minor are the same, don't notify
		return false
	}

	published, err := time.Parse(time.RFC3339, result.PublishedAt)
	if err != nil {
		log.Printf("unable to parse publish date: %v", err)
		return false
	}

	if published.UTC().Add(24 * time.Hour).Before(time.Now().UTC()) {
		return true
	}

	return false
}
