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

type response struct {
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
	HandleFatalError("Failed to get launcher information", err, hub)
	defer body.Close()
	updateProgress(1)

	var res response
	err = json.NewDecoder(body).Decode(&res)
	HandleFatalError("Failed to deserialize launcher information", err, hub)
	updateProgress(1)

	if FileExists(targetPath) {
		if file, err := os.Open(targetPath); err == nil {
			defer file.Close()
			sha := sha1.New()
			if _, err := io.Copy(sha, file); err == nil {
				hash := hex.EncodeToString(sha.Sum(nil))
				if hash == res.Hash {
					updateProgress(3)
					return
				}
			}
		}
	}
	updateProgress(1)

	HandleFatalError("Failed to create launcher directories", err, hub)
	updateProgress(1)

	err = DownloadFromUrl(res.Url, targetPath)
	HandleFatalError("Failed to download launcher", err, hub)
	updateProgress(1)
}

func BeginJre(wg *sync.WaitGroup) {
	defer wg.Done()

	hub := CreateSentryHub("BeginJre")

	basePath := filepath.Join(WorkingDir, "jre", "17")
	manifestPath := filepath.Join(basePath, "version.json")
	err := os.MkdirAll(basePath, os.ModePerm)
	HandleFatalError("Failed to create JRE directories", err, hub)
	updateProgress(1)

	body, err := GetFromUrl(fmt.Sprintf("%s/jre?version=17&os=%s&arch=%s", MetadataURL, Sys, Arch))
	HandleFatalError("Failed to get JRE information", err, hub)
	defer body.Close()
	updateProgress(1)

	var res response
	err = json.NewDecoder(body).Decode(&res)
	HandleFatalError("Failed to deserialize JRE information", err, hub)
	updateProgress(1)

	if FileExists(manifestPath) {
		var manifest jreManifest
		if bytes, err := os.ReadFile(manifestPath); err == nil {
			if err := json.Unmarshal(bytes, &manifest); err == nil {
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
	HandleFatalError("Failed to download JRE", err, hub)
	updateProgress(1)

	extractedPath := filepath.Join(basePath, "extracted")
	err = os.RemoveAll(extractedPath)
	HandleFatalError("Failed to delete old JRE", err, hub)
	zipArchiver := &archiver.Zip{StripComponents: 1}
	err = zipArchiver.Unarchive(targetPath, extractedPath)
	HandleFatalError("Failed to unzip JRE", err, hub)
	updateProgress(1)

	manifest := jreManifest{Hash: res.Hash, Size: res.Size}
	bytes, err := json.Marshal(manifest)
	HandleFatalError("Failed to serialize JRE manifest", err, hub)
	err = os.WriteFile(manifestPath, bytes, os.ModePerm)
	HandleFatalError("Failed to write JRE manifest", err, hub)
	updateProgress(1)

	err = os.Remove(targetPath)
	HandleFatalError("Failed to delete JRE zip", err, hub)
	updateProgress(1)
}

func updateProgress(steps int) {
	CompletedTasks += steps
	giu.Update()
}
