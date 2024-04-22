package main

import (
	"archive/zip"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/alpine-client/pinnacle/sentry"
	"github.com/alpine-client/pinnacle/ui"
)

type MetadataResponse struct {
	URL  string `json:"url"`
	Hash string `json:"sha1"`
	Size uint32 `json:"size"`
}

type JreManifest struct {
	Hash string `json:"checksum"`
	Size uint32 `json:"size"`
}

func runTasks(done chan bool) {
	tasks := []struct {
		name string
		task func(context.Context) error
	}{
		{"checkJRE", checkJRE},
		{"checkLauncher", checkLauncher},
		{"runLauncher", runLauncher},
	}

	for _, t := range tasks {
		ctx := sentry.NewContext(t.name)
		err := t.task(ctx)
		if err != nil {
			ui.Close()
			ui.DisplayError(ctx, err)
			break
		}
	}

	done <- true
}

func fetchMetadata(ctx context.Context, url string) (*MetadataResponse, error) {
	body, err := getFromURL(ctx, url)
	if err != nil {
		return nil, err
	}
	defer func() {
		sentry.CaptureErr(ctx, body.Close())
	}()
	sentry.Breadcrumb(ctx, "decoding response from "+url)
	var res MetadataResponse
	if err = json.NewDecoder(body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}

func fileHashMatches(ctx context.Context, hash string, path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() {
		sentry.CaptureErr(ctx, file.Close())
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

func checkLauncher(ctx context.Context) error {
	ui.UpdateProgress(5, "Validating launcher...")
	sentry.Breadcrumb(ctx, "fetching metadata from /pinnacle")

	launcher, err := fetchMetadata(ctx, MetadataURL+"/pinnacle")
	if err != nil {
		return err
	}

	targetPath := alpinePath("launcher.jar")
	if !fileExists(targetPath) {
		sentry.Breadcrumb(ctx, "missing launcher.jar")
		goto DOWNLOAD
	}

	if !fileHashMatches(ctx, launcher.Hash, targetPath) {
		sentry.Breadcrumb(ctx, "failed checksum validation")
		goto DOWNLOAD
	}

	ui.UpdateProgress(25, "Preparing to start launcher...")
	sentry.Breadcrumb(ctx, "finished checkLauncher (jar existed)")
	return nil

DOWNLOAD:
	err = downloadLauncher(ctx, launcher, targetPath)
	if err != nil {
		return err
	}
	sentry.Breadcrumb(ctx, "finished checkLauncher (jar downloaded)")
	return nil
}

func downloadLauncher(ctx context.Context, manifest *MetadataResponse, dest string) error {
	err := downloadFromURL(ctx, manifest.URL, dest)
	if err != nil {
		return err
	}
	ui.UpdateProgress(20, "Verifying launcher hash...")
	if !fileHashMatches(ctx, manifest.Hash, dest) {
		sentry.Breadcrumb(ctx, "failed checksum validation after download", sentry.LevelError)
		return errors.New("fatal error")
	}
	ui.UpdateProgress(5, "Preparing to start launcher...")
	return nil
}

func checkJRE(ctx context.Context) error {
	path := alpinePath("jre", "17")

	sentry.Breadcrumb(ctx, "mkdir "+path)
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}
	ui.UpdateProgress(5, "Fetching metadata...")

	endpoint := fmt.Sprintf("%s/jre?version=17&os=%s&arch=%s", MetadataURL, Sys, Arch)
	sentry.Breadcrumb(ctx, "fetching manifest from "+endpoint)
	jre, err := fetchMetadata(ctx, endpoint)
	if err != nil {
		return err
	}
	ui.UpdateProgress(15, "Validating java...")

	var data []byte
	var manifest JreManifest
	manifestPath := alpinePath("jre", "17", "version.json")
	if !fileExists(manifestPath) {
		sentry.Breadcrumb(ctx, "missing manifest")
		goto DOWNLOAD
	}
	ui.UpdateProgress(2)

	data, err = os.ReadFile(manifestPath)
	if err != nil {
		sentry.Breadcrumb(ctx, "failed to read manifest file")
		goto DOWNLOAD
	}
	ui.UpdateProgress(2)

	if err = json.Unmarshal(data, &manifest); err != nil {
		sentry.Breadcrumb(ctx, "failed to unmarshal manifest file")
		goto DOWNLOAD
	}
	ui.UpdateProgress(2)

	if manifest.Hash != jre.Hash {
		sentry.Breadcrumb(ctx, fmt.Sprintf("checksum from file %s does not match expected %s", manifest.Hash, jre.Hash))
		goto DOWNLOAD
	}

	ui.UpdateProgress(340)
	sentry.Breadcrumb(ctx, "finished checkJRE (existed)")
	return nil

DOWNLOAD:
	err = downloadJRE(ctx, jre)
	if err != nil {
		return err
	}
	sentry.Breadcrumb(ctx, "finished checkJRE (downloaded)")
	return nil
}

func downloadJRE(ctx context.Context, m *MetadataResponse) error {
	zipPath := alpinePath("jre", "17", "jre.zip")

	ui.UpdateProgress(1, "Downloading java...")
	err := downloadFromURL(ctx, m.URL, zipPath)
	if err != nil {
		return err
	}
	ui.UpdateProgress(20)

	extractedPath := alpinePath("jre", "17", "extracted")
	sentry.Breadcrumb(ctx, "cleaning up path "+extractedPath)
	err = os.RemoveAll(extractedPath)
	if err != nil {
		return err
	}
	ui.UpdateProgress(5)

	sentry.Breadcrumb(ctx, "extracting zip "+zipPath)
	err = unzipAll(ctx, zipPath, extractedPath)
	if err != nil {
		return err
	}

	bytes, err := json.Marshal(JreManifest{Hash: m.Hash, Size: m.Size})
	if err != nil {
		return err
	}
	ui.UpdateProgress(5)

	manifestPath := alpinePath("jre", "17", "version.json")
	sentry.Breadcrumb(ctx, "writing manifest to file "+manifestPath)
	err = os.WriteFile(manifestPath, bytes, os.ModePerm)
	if err != nil {
		return err
	}
	ui.UpdateProgress(5)

	_ = os.Remove(zipPath) // failing to delete old zip won't break anything.
	sentry.Breadcrumb(ctx, "finished checkJRE (downloaded)")
	ui.UpdateProgress(5)
	return nil
}

func runLauncher(ctx context.Context) error {
	jarPath := alpinePath("launcher.jar")
	jrePath := alpinePath("jre", "17", "extracted", "bin", Sys.javaExecutable())

	ui.UpdateProgress(1, "Starting launcher...")

	args := []string{
		"-Xms256M",
		"-Xmx256M",
	}

	if Sys == Mac {
		args = append(args, "-XstartOnFirstThread")
	}

	args = append(args, "-jar", jarPath)

	if version != "" {
		args = append(args, "--pinnacle-version", version)
	}

	processAttr := &os.ProcAttr{
		Dir:   alpinePath(),
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	}

	sentry.Breadcrumb(ctx, fmt.Sprintf("starting launcher process: %s %s", jrePath, args))
	proc, err := os.StartProcess(jrePath, args, processAttr)
	if err != nil {
		return err
	}
	ui.UpdateProgress(20)

	sentry.Breadcrumb(ctx, "releasing launcher process")
	err = proc.Release()
	if err != nil {
		return err
	}
	ui.UpdateProgress(int(ui.TotalSteps))
	return nil
}

func unzipAll(ctx context.Context, src string, dest string) error {
	zipReader, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		sentry.CaptureErr(ctx, zipReader.Close())
	}()
	if err = os.MkdirAll(dest, os.ModePerm); err != nil {
		return err
	}
	count := len(zipReader.File)
	for i, file := range zipReader.File {
		ui.UpdateProgress(1, fmt.Sprintf("Extracting java (%d/%d)...", i, count))
		parts := strings.Split(file.Name, "/")
		if len(parts) > 1 {
			parts = parts[1:]
		}
		fPath := filepath.Join(dest, filepath.Join(parts...))
		if file.FileInfo().IsDir() {
			if err = os.MkdirAll(fPath, os.ModePerm); err != nil {
				return err
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
			sentry.CaptureErr(ctx, outFile.Close())
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
	}
	return nil
}
