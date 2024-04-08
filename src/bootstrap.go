package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/AllenDang/giu"
	"github.com/mholt/archiver/v3"
)

const TotalTasks = 13

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
	hub := CreateSentryHub("FetchMetadata")
	body, err := GetFromUrl(url)
	CaptureAndExit(err, hub)
	defer body.Close()

	var res MetadataResponse
	err = json.NewDecoder(body).Decode(&res)
	CaptureAndExit(err, hub)
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
	defer wg.Done()

	hub := CreateSentryHub("BeginLauncher")
	updateProgress(1)
	launcher := FetchMetadata(MetadataURL + "/pinnacle")
	updateProgress(1)

	targetPath := filepath.Join(WorkingDir, "launcher.jar")
	if !FileExists(targetPath) || !FileHashMatches(launcher.Hash, targetPath) {
		updateProgress(1)
		err := DownloadFromUrl(launcher.Url, targetPath)
		CaptureAndExit(err, hub)
	}
	updateProgress(1)
	return
}

func BeginJre(wg *sync.WaitGroup) {
	defer wg.Done()

	hub := CreateSentryHub("BeginJre")
	basePath := filepath.Join(WorkingDir, "jre", "17")

	err := os.MkdirAll(basePath, os.ModePerm)
	CaptureAndExit(err, hub)
	updateProgress(1)

	CaptureAndExit(err, hub)
	jre := FetchMetadata(fmt.Sprintf("%s/jre?version=17&os=%s&arch=%s", MetadataURL, Sys, Arch))
	updateProgress(2)

	manifestPath := filepath.Join(basePath, "version.json")
	if FileExists(manifestPath) {
		var manifest jreManifest
		var bytes []byte
		if bytes, err = os.ReadFile(manifestPath); err == nil {
			if err = json.Unmarshal(bytes, &manifest); err == nil {
				if manifest.Hash == jre.Hash {
					updateProgress(5)
					return
				}
			}
		}
	}
	updateProgress(1)

	targetPath := filepath.Join(basePath, "jre.zip")
	err = DownloadFromUrl(jre.Url, targetPath)
	CaptureAndExit(err, hub)
	updateProgress(1)

	extractedPath := filepath.Join(basePath, "extracted")
	err = os.RemoveAll(extractedPath)
	CaptureAndExit(err, hub)
	zipArchiver := &archiver.Zip{StripComponents: 1, OverwriteExisting: true}
	err = zipArchiver.Unarchive(targetPath, extractedPath)
	CaptureAndExit(err, hub)
	updateProgress(1)

	bytes, err := json.Marshal(jreManifest{Hash: jre.Hash, Size: jre.Size})
	CaptureAndExit(err, hub)

	err = os.WriteFile(manifestPath, bytes, os.ModePerm)
	CaptureAndExit(err, hub)
	updateProgress(1)

	// We can safely ignore this error; failing to delete old zip won't break anything.
	_ = os.Remove(targetPath)
	updateProgress(1)
}

func updateProgress(steps int) {
	CompletedTasks += steps
	giu.Update()
}
