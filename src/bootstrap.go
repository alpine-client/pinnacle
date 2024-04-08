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

type metadataResponse struct {
	Url  string `json:"url"`
	Hash string `json:"sha1"`
	Size uint32 `json:"size"`
}

type jreManifest struct {
	Hash string `json:"checksum"`
	Size uint32 `json:"size"`
}

func BeginLauncher(wg *sync.WaitGroup) {
	defer wg.Done()

	hub := CreateSentryHub("BeginLauncher")
	targetPath := filepath.Join(WorkingDir, "launcher.jar")
	body, err := GetFromUrl(MetadataURL + "/pinnacle")
	CaptureAndExit(err, hub)
	defer body.Close()
	updateProgress(1)

	var res metadataResponse
	err = json.NewDecoder(body).Decode(&res)
	CaptureAndExit(err, hub)
	updateProgress(1)

	if FileExists(targetPath) {
		var file *os.File
		if file, err = os.Open(targetPath); err == nil {
			defer file.Close()
			sha := sha1.New()
			_, _ = io.Copy(sha, file)
			if hex.EncodeToString(sha.Sum(nil)) == res.Hash {
				return
			}
		}

	}
	updateProgress(1)
	err = DownloadFromUrl(res.Url, targetPath)
	CaptureAndExit(err, hub)
	updateProgress(1)
}

func BeginJre(wg *sync.WaitGroup) {
	defer wg.Done()

	hub := CreateSentryHub("BeginJre")

	basePath := filepath.Join(WorkingDir, "jre", "17")
	manifestPath := filepath.Join(basePath, "version.json")
	err := os.MkdirAll(basePath, os.ModePerm)
	CaptureAndExit(err, hub)
	updateProgress(1)

	body, err := GetFromUrl(fmt.Sprintf("%s/jre?version=17&os=%s&arch=%s", MetadataURL, Sys, Arch))
	CaptureAndExit(err, hub)
	defer body.Close()
	updateProgress(1)

	var res metadataResponse
	err = json.NewDecoder(body).Decode(&res)
	CaptureAndExit(err, hub)
	updateProgress(1)

	if FileExists(manifestPath) {
		var manifest jreManifest
		var bytes []byte
		if bytes, err = os.ReadFile(manifestPath); err == nil {
			if err = json.Unmarshal(bytes, &manifest); err == nil {
				if manifest.Hash == res.Hash {
					updateProgress(5)
					return
				}
			}
		}
	}
	updateProgress(1)

	targetPath := filepath.Join(basePath, "jre.zip")
	err = DownloadFromUrl(res.Url, targetPath)
	CaptureAndExit(err, hub)
	updateProgress(1)

	extractedPath := filepath.Join(basePath, "extracted")
	err = os.RemoveAll(extractedPath)
	CaptureAndExit(err, hub)
	zipArchiver := &archiver.Zip{StripComponents: 1, OverwriteExisting: true}
	err = zipArchiver.Unarchive(targetPath, extractedPath)
	CaptureAndExit(err, hub)
	updateProgress(1)

	bytes, err := json.Marshal(jreManifest{Hash: res.Hash, Size: res.Size})
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
