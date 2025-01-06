package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alpine-client/pinnacle/sentry"
	"github.com/alpine-client/pinnacle/ui"
)

var httpClient = &http.Client{}

func (p *Pinnacle) getFromURL(ctx context.Context, url string) (*http.Response, error) {
	const maxAttempts = 4
	var statusCode int

	for i := range maxAttempts {
		if i > 0 {
			<-time.After(time.Second * time.Duration(2<<i)) // Exponential backoff
		}
		p.Breadcrumb(ctx, fmt.Sprintf("[%d] making request to %s", i+1, url))

		request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("User-Agent", fmt.Sprintf("Pinnacle/%s (%s; %s)", version, p.os, p.arch))

		response, err := httpClient.Do(request)
		if err != nil {
			sentry.Breadcrumb(ctx, fmt.Sprintf("[%d] request error: %v", i+1, err), slog.LevelError)
			continue
		}

		statusCode = response.StatusCode
		p.Breadcrumb(ctx, fmt.Sprintf("[%d] status code: %d", i+1, statusCode))
		if statusCode == http.StatusOK {
			return response, nil
		}

		err = response.Body.Close()
		if err != nil {
			p.Breadcrumb(ctx, fmt.Sprintf("[%d] failed to close body: %v", i+1, err), slog.LevelError)
		}
	}
	return nil, errors.New("internet failure")
}

func (p *Pinnacle) downloadFile(ctx context.Context, url string, path string, pt *ui.ProgressiveTask) error {
	resp, err := p.getFromURL(ctx, url)
	if err != nil {
		return err
	}
	defer func() {
		p.CaptureErr(ctx, resp.Body.Close())
	}()

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		p.CaptureErr(ctx, file.Close())
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
			if !errors.Is(er, io.EOF) {
				err = er
			}
			break
		}
	}
	return err
}

func (p *Pinnacle) fetchSentryDSN() string {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const endpoint = MetadataURL + "/sentry"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		p.logger.Warn(err.Error())
		return ""
	}
	req.Header.Set("User-Agent", fmt.Sprintf("alpine-client/pinnacle/%s (%s)", version, SupportEmail))

	resp, err := httpClient.Do(req)
	if err != nil {
		p.logger.Warn(err.Error())
		return ""
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			p.logger.Warn("unable to close response body", slog.Any("error", err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		p.logger.Warn(fmt.Sprintf("unable to fetch sentry DSN: status code %d", resp.StatusCode))
		return ""
	}

	type response struct {
		DSN string `json:"dsn"`
	}

	var result response
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		p.logger.Warn(err.Error())
		return ""
	}

	return result.DSN
}

func (p *Pinnacle) isUpdateAvailable(c context.Context) bool {
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
			p.logger.WarnContext(c, err.Error())
		}
	}()

	type details struct {
		TagName     string `json:"tag_name"`
		PublishedAt string `json:"published_at"`
		PreRelease  bool   `json:"prerelease"`
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
		p.logger.WarnContext(c, err.Error())
		return false
	}

	if published.UTC().Add(24 * time.Hour).Before(time.Now().UTC()) {
		return true
	}

	return false
}
