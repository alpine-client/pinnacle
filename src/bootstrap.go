package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mholt/archiver/v3"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/AllenDang/giu"
)

const TotalTasks = 10

var CompletedTasks = 0

type MetadataResponse struct {
	Url  string `json:"url"`
	Hash string `json:"sha1"`
	Size uint32 `json:"size"`
}

type jreManifest struct {
	Hash string `json:"checksum"`
	Size uint32 `json:"size"`
}

func FetchMetadata(url string) *MetadataResponse {
	ctx := CreateSentryCtx("FetchMetadata")
	body, err := GetFromUrl(url)
	CrumbCaptureExit(ctx, err, "making request to "+url)
	defer body.Close()

	var res MetadataResponse
	err = json.NewDecoder(body).Decode(&res)
	CrumbCaptureExit(ctx, err, "decoding response from "+url)
	return &res
}

func FileHashMatches(hash string, path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	sha := sha1.New()
	if _, err = io.Copy(sha, file); err != nil {
		return false
	}
	if hex.EncodeToString(sha.Sum(nil)) == hash {
		return true
	}
	return false
}

func BeginLauncher(wg *sync.WaitGroup) {
	ctx := CreateSentryCtx("BeginLauncher")
	launcher := FetchMetadata(MetadataURL + "/pinnacle")
	AddBreadcrumb(ctx, "fetched metadata from /pinnacle")
	updateProgress(1)
	targetPath := filepath.Join(WorkingDir, "launcher.jar")
	if !FileExists(targetPath) || !FileHashMatches(launcher.Hash, targetPath) {
		updateProgress(1)
		err := DownloadFromUrl(launcher.Url, targetPath)
		CrumbCaptureExit(ctx, err, "downloading from "+launcher.Url)
		updateProgress(1)
		if !FileHashMatches(launcher.Hash, targetPath) {
			CrumbCaptureExit(ctx, errors.New("fatal error"), "failed checksum validation after download")
		}
		AddBreadcrumb(ctx, "finished BeginLauncher (jar downloaded)")
	} else {
		updateProgress(2)
		AddBreadcrumb(ctx, "finished (jar existed)")
	}
	wg.Done()
	return
}

func BeginJre(wg *sync.WaitGroup) {
	ctx := CreateSentryCtx("BeginJre")
	basePath := filepath.Join(WorkingDir, "jre", "17")

	err := os.MkdirAll(basePath, os.ModePerm)
	CrumbCaptureExit(ctx, err, "mkdir "+basePath)
	updateProgress(1)

	url := fmt.Sprintf("%s/jre?version=17&os=%s&arch=%s", MetadataURL, Sys, Arch)
	jre := FetchMetadata(url)
	AddBreadcrumb(ctx, "fetched manifest from "+url)
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

	updateProgress(4)
	AddBreadcrumb(ctx, "finished BeginJre (existed)")
	wg.Done()
	return

DOWNLOAD:
	DownloadJRE(ctx, jre)
	updateProgress(1)
	AddBreadcrumb(ctx, "finished BeginJre (downloaded)")
	wg.Done()
	return
}

func DownloadJRE(ctx context.Context, m *MetadataResponse) {
	basePath := filepath.Join(WorkingDir, "jre", "17")
	manifestPath := filepath.Join(basePath, "version.json")
	targetPath := filepath.Join(basePath, "jre.zip")

	err := DownloadFromUrl(m.Url, targetPath)
	CrumbCaptureExit(ctx, err, "downloading from "+m.Url)
	updateProgress(1)

	extractedPath := filepath.Join(basePath, "extracted")
	err = os.RemoveAll(extractedPath)
	CrumbCaptureExit(ctx, err, "cleaning up path: "+extractedPath)
	updateProgress(1)

	zipArchiver := &archiver.Zip{StripComponents: 1, OverwriteExisting: true}
	err = zipArchiver.Unarchive(targetPath, extractedPath)
	CrumbCaptureExit(ctx, err, "extracting zip")
	updateProgress(1)

	bytes, err := json.Marshal(jreManifest{Hash: m.Hash, Size: m.Size})
	CrumbCaptureExit(ctx, err, "marshaling manifest")

	err = os.WriteFile(manifestPath, bytes, os.ModePerm)
	CrumbCaptureExit(ctx, err, "writing manifest to file")
	updateProgress(1)

	// We can safely ignore this error; failing to delete old zip won't break anything.
	_ = os.Remove(targetPath)
	AddBreadcrumb(ctx, "finished BeginJre (downloaded)")
}

var mutex sync.Mutex

func updateProgress(steps int) {
	mutex.Lock()
	CompletedTasks += steps
	giu.Update()
	mutex.Unlock()
}
