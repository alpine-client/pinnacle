package main

import (
	"archive/zip"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/getsentry/sentry-go"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/AllenDang/giu"
)

const TotalTasks = 43

var CompletedTasks = 0

type MetadataResponse struct {
	URL  string `json:"url"`
	Hash string `json:"sha1"`
	Size uint32 `json:"size"`
}

type jreManifest struct {
	Hash string `json:"checksum"`
	Size uint32 `json:"size"`
}

func FetchMetadata(ctx context.Context, url string) (*MetadataResponse, error) {
	body, err := GetFromURL(ctx, url)
	if err != nil {
		return nil, err
	}
	defer func() {
		CaptureErrExit(ctx, body.Close())
	}()
	AddBreadcrumb(ctx, "decoding response from "+url)
	var res MetadataResponse
	if err = json.NewDecoder(body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}

func FileHashMatches(ctx context.Context, hash string, path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() {
		if err = file.Close(); err != nil {
			CaptureErrExit(ctx, err)
		}
	}()
	sha := sha1.New()
	if _, err = io.Copy(sha, file); err != nil {
		return false
	}
	if hex.EncodeToString(sha.Sum(nil)) == hash {
		return true
	}
	return false
}

func BeginLauncher() {
	ctx := CreateSentryCtx("BeginLauncher")

	AddBreadcrumb(ctx, "fetching metadata from /pinnacle")
	launcher, err := FetchMetadata(ctx, MetadataURL+"/pinnacle")
	CaptureErrExit(ctx, err)
	updateProgress(1)

	targetPath := filepath.Join(WorkingDir, "launcher.jar")
	if !FileExists(targetPath) {
		AddBreadcrumb(ctx, "missing launcher.jar")
		goto DOWNLOAD
	}
	updateProgress(1)

	if !FileHashMatches(ctx, launcher.Hash, targetPath) {
		AddBreadcrumb(ctx, "failed checksum validation")
		goto DOWNLOAD
	}
	updateProgress(1)
	AddBreadcrumb(ctx, "finished BeginLauncher (jar existed)")

DOWNLOAD:
	DownloadLauncher(ctx, launcher, targetPath)
	updateProgress(1)
	AddBreadcrumb(ctx, "finished BeginLauncher (jar downloaded)")
}

func DownloadLauncher(ctx context.Context, manifest *MetadataResponse, dest string) {
	err := DownloadFromURL(ctx, manifest.URL, dest)
	CaptureErrExit(ctx, err)
	updateProgress(1)
	if !FileHashMatches(ctx, manifest.Hash, dest) {
		AddBreadcrumb(ctx, "failed checksum validation after download", sentry.LevelError)
		CaptureErrExit(ctx, errors.New("fatal error"))
	}
}

func BeginJre() {
	ctx := CreateSentryCtx("BeginJre")
	basePath := filepath.Join(WorkingDir, "jre", "17")

	AddBreadcrumb(ctx, "mkdir "+basePath)
	err := os.MkdirAll(basePath, os.ModePerm)
	CaptureErrExit(ctx, err)
	updateProgress(1)

	URL := fmt.Sprintf("%s/jre?version=17&os=%s&arch=%s", MetadataURL, Sys, Arch)
	AddBreadcrumb(ctx, "fetching manifest from "+URL)
	jre, err := FetchMetadata(ctx, URL)
	CaptureErrExit(ctx, err)
	updateProgress(1)

	var data []byte
	var manifest jreManifest
	manifestPath := filepath.Join(basePath, "version.json")
	if !FileExists(manifestPath) {
		AddBreadcrumb(ctx, "missing manifest")
		goto DOWNLOAD
	}

	updateProgress(1)
	data, err = os.ReadFile(manifestPath)
	if err != nil {
		AddBreadcrumb(ctx, "failed to read manifest file")
		goto DOWNLOAD
	}

	if err = json.Unmarshal(data, &manifest); err != nil {
		AddBreadcrumb(ctx, "failed to unmarshal manifest file")
		goto DOWNLOAD
	}

	if manifest.Hash != jre.Hash {
		AddBreadcrumb(ctx, fmt.Sprintf("checksum from file %s does not match expected %s", manifest.Hash, jre.Hash))
		goto DOWNLOAD
	}

	updateProgress(34)
	AddBreadcrumb(ctx, "finished BeginJre (existed)")
	return

DOWNLOAD:
	DownloadJRE(ctx, jre)
	updateProgress(1)
	AddBreadcrumb(ctx, "finished BeginJre (downloaded)")
}

func DownloadJRE(ctx context.Context, m *MetadataResponse) {
	basePath := filepath.Join(WorkingDir, "jre", "17")
	manifestPath := filepath.Join(basePath, "version.json")
	zipPath := filepath.Join(basePath, "jre.zip")

	err := DownloadFromURL(ctx, m.URL, zipPath)
	CaptureErrExit(ctx, err)
	updateProgress(1)

	extractedPath := filepath.Join(basePath, "extracted")
	AddBreadcrumb(ctx, "cleaning up path "+extractedPath)
	err = os.RemoveAll(extractedPath)
	CaptureErrExit(ctx, err)
	updateProgress(1)

	AddBreadcrumb(ctx, "extracting zip "+zipPath)
	err = Unzip(ctx, zipPath, extractedPath)
	CaptureErrExit(ctx, err)
	updateProgress(1)

	bytes, err := json.Marshal(jreManifest{Hash: m.Hash, Size: m.Size})
	CaptureErrExit(ctx, err)

	AddBreadcrumb(ctx, "writing manifest to file "+manifestPath)
	err = os.WriteFile(manifestPath, bytes, os.ModePerm)
	CaptureErrExit(ctx, err)
	updateProgress(1)

	// We can safely ignore this error; failing to delete old zip won't break anything.
	_ = os.Remove(zipPath)
	AddBreadcrumb(ctx, "finished BeginJre (downloaded)")
}

func Unzip(ctx context.Context, src string, dest string) error {
	zipReader, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err = zipReader.Close(); err != nil {
			CaptureErrExit(ctx, err)
		}
	}()
	if err = os.MkdirAll(dest, os.ModePerm); err != nil {
		return err
	}
	for i, file := range zipReader.File {
		parts := strings.Split(file.Name, "/")
		if len(parts) > 1 {
			parts = parts[1:]
		}
		fPath := filepath.Join(dest, filepath.Join(parts...))
		if file.FileInfo().IsDir() {
			if err = os.MkdirAll(fPath, os.ModePerm); err != nil {
				return err
			}
			if i%10 == 0 {
				updateProgress(1)
			}
			continue
		}
		if err = os.MkdirAll(filepath.Dir(fPath), os.ModePerm); err != nil {
			return err
		}
		var outFile *os.File
		outFile, err = os.OpenFile(fPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		var rc io.ReadCloser
		rc, err = file.Open()
		if err != nil {
			if closeErr := outFile.Close(); closeErr != nil {
				CaptureErrExit(ctx, closeErr)
			}
			return err
		}
		if _, err = io.Copy(outFile, rc); err != nil {
			return err
		}
		if err = outFile.Close(); err != nil {
			return err
		}
		if err = rc.Close(); err != nil {
			return err
		}

		if i%10 == 0 {
			updateProgress(1)
		}
	}
	return nil
}

var mutex sync.Mutex

func updateProgress(steps int) {
	mutex.Lock()
	CompletedTasks += steps
	giu.Update()
	mutex.Unlock()
}
