package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"

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

type TaskResult struct {
	ctx context.Context
	err error
}

func runTasks(done chan bool) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	results := make(chan TaskResult, 2)

	results <- checkJRE(ctx)
	results <- checkLauncher(ctx)

	// Check errors from both tasks
	var res TaskResult
	var failed *TaskResult
	for range 2 {
		res = <-results
		if res.err != nil {
			failed = &res
			cancel()
			break
		}
	}
	close(results)

	if failed != nil {
		cleanup()
		ui.CloseSplash()
		ui.DisplayError(failed.ctx, failed.err)
	} else {
		start := runLauncher(ctx)
		if start.err != nil {
			cleanup()
		}
		ui.CloseSplash()
		ui.DisplayError(start.ctx, start.err)
	}

	done <- true
}

func fetchMetadata(ctx context.Context, url string) (*MetadataResponse, error) {
	resp, err := getFromURL(ctx, url)
	if err != nil {
		return nil, err
	}

	defer func() {
		sentry.CaptureErr(ctx, resp.Body.Close())
	}()

	sentry.Breadcrumb(ctx, "decoding response from "+url)
	var res MetadataResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func fileHashMatches(ctx context.Context, hash string, path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer func() {
		sentry.CaptureErr(ctx, file.Close())
	}()

	sha := sha1.New()
	_, err = io.Copy(sha, file)
	if err != nil {
		return false, err
	}

	result := hex.EncodeToString(sha.Sum(nil))
	if result == hash {
		return true, nil
	}

	return false, fmt.Errorf("hash mismatch: got %s expected %s", result, hash)
}

func checkLauncher(c context.Context) TaskResult {
	pt := ui.NewProgressTask("Preparing launcher...")
	ctx := sentry.NewContext(c, "checkLauncher")

	sentry.Breadcrumb(ctx, "fetching metadata from /pinnacle")
	pt.UpdateProgress(0.20)

	launcher, err := fetchMetadata(ctx, MetadataURL+"/pinnacle")
	if err != nil {
		return TaskResult{ctx, err}
	}
	pt.UpdateProgress(0.60)

	targetPath := alpinePath("launcher.jar")
	if !fileExists(targetPath) {
		sentry.Breadcrumb(ctx, "missing launcher.jar")
		pt.Close()
		goto DOWNLOAD
	}

	if validHash, _ := fileHashMatches(ctx, launcher.Hash, targetPath); !validHash {
		sentry.Breadcrumb(ctx, "failed checksum validation")
		pt.Close()
		goto DOWNLOAD
	}

	pt.Close()
	sentry.Breadcrumb(ctx, "finished checkLauncher (jar existed)")
	return TaskResult{ctx, nil}

DOWNLOAD:
	err = downloadLauncher(ctx, launcher, targetPath)
	if err != nil {
		return TaskResult{ctx, err}
	}
	sentry.Breadcrumb(ctx, "finished checkLauncher (jar downloaded)")
	return TaskResult{ctx, nil}
}

func downloadLauncher(ctx context.Context, manifest *MetadataResponse, dest string) error {
	pt := ui.NewProgressTask("Downloading launcher...")
	err := downloadFile(ctx, manifest.URL, dest, pt)
	if err != nil {
		return err
	}

	var validHash bool
	if validHash, err = fileHashMatches(ctx, manifest.Hash, dest); !validHash {
		sentry.Breadcrumb(ctx, fmt.Sprintf("hash mismatch after download (retrying): %v", err), sentry.LevelError)

		_ = os.RemoveAll(dest)
		err = downloadFile(ctx, manifest.URL, dest, pt)
		if err != nil {
			return err
		}

		if validHash, err = fileHashMatches(ctx, manifest.Hash, dest); !validHash {
			sentry.Breadcrumb(ctx, fmt.Sprintf("hash mismatch after download (fatal): %v", err), sentry.LevelError)
			return err
		}
	}

	pt.UpdateProgress(0.99999, "Starting launcher...")
	return nil
}

func checkJRE(c context.Context) TaskResult {
	pt := ui.NewProgressTask("Preparing java runtime...")
	ctx := sentry.NewContext(c, "checkJRE")
	path := alpinePath("jre", "17")

	sentry.Breadcrumb(ctx, "mkdir "+path)
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return TaskResult{ctx, err}
	}
	pt.UpdateProgress(0.20)

	endpoint := fmt.Sprintf("%s/jre?version=17&os=%s&arch=%s", MetadataURL, Sys, Arch)
	sentry.Breadcrumb(ctx, "fetching manifest from "+endpoint)
	jre, err := fetchMetadata(ctx, endpoint)
	if err != nil {
		return TaskResult{ctx, err}
	}

	var data []byte
	var manifest JreManifest
	javaPath := alpinePath("jre", "17", "extracted", "bin", Sys.javaExecutable())
	manifestPath := alpinePath("jre", "17", "version.json")
	pt.UpdateProgress(0.50)

	if !fileExists(javaPath) {
		sentry.Breadcrumb(ctx, "missing java executable")
		pt.Close()
		goto DOWNLOAD
	}
	pt.UpdateProgress(0.68)

	if !fileExists(manifestPath) {
		sentry.Breadcrumb(ctx, "missing manifest")
		pt.Close()
		goto DOWNLOAD
	}
	pt.UpdateProgress(0.76)

	data, err = os.ReadFile(manifestPath)
	if err != nil {
		sentry.Breadcrumb(ctx, "failed to read manifest file")
		pt.Close()
		goto DOWNLOAD
	}
	pt.UpdateProgress(0.85)

	if err = json.Unmarshal(data, &manifest); err != nil {
		sentry.Breadcrumb(ctx, "failed to unmarshal manifest file")
		pt.Close()
		goto DOWNLOAD
	}
	pt.UpdateProgress(0.98)

	if manifest.Hash != jre.Hash {
		sentry.Breadcrumb(ctx, fmt.Sprintf("checksum from file %s does not match expected %s", manifest.Hash, jre.Hash))
		pt.Close()
		goto DOWNLOAD
	}

	pt.Close()
	sentry.Breadcrumb(ctx, "finished checkJRE (existed)")
	return TaskResult{ctx, nil}

DOWNLOAD:
	err = downloadJRE(ctx, jre)
	if err != nil {
		return TaskResult{ctx, nil}
	}
	sentry.Breadcrumb(ctx, "finished checkJRE (downloaded)")
	return TaskResult{ctx, nil}
}

func downloadJRE(ctx context.Context, m *MetadataResponse) error {
	zipPath := alpinePath("jre", "17", "jre.zip")

	pt := ui.NewProgressTask("Downloading java...")
	err := downloadFile(ctx, m.URL, zipPath, pt)
	if err != nil {
		return err
	}
	pt.Close()

	pt = ui.NewProgressTask("Extracting java...")
	extractedPath := alpinePath("jre", "17", "extracted")
	sentry.Breadcrumb(ctx, "cleaning up path "+extractedPath)
	err = os.RemoveAll(extractedPath)
	if err != nil {
		return err
	}

	sentry.Breadcrumb(ctx, "extracting zip "+zipPath)
	err = extractAll(ctx, zipPath, extractedPath, pt)
	if err != nil {
		return err
	}

	_ = os.Chmod(alpinePath("jre", "17", "extracted", "bin", Sys.javaExecutable()), 0o755)

	bytes, err := json.Marshal(JreManifest{Hash: m.Hash, Size: m.Size})
	if err != nil {
		return err
	}

	manifestPath := alpinePath("jre", "17", "version.json")
	sentry.Breadcrumb(ctx, "writing manifest to file "+manifestPath)
	err = os.WriteFile(manifestPath, bytes, 0o600)
	if err != nil {
		return err
	}

	_ = os.Remove(zipPath)
	sentry.Breadcrumb(ctx, "finished checkJRE (downloaded)")
	pt.Close()
	return nil
}

func runLauncher(c context.Context) TaskResult {
	ctx := sentry.NewContext(c, "runLauncher")

	jarPath := alpinePath("launcher.jar")
	jrePath := alpinePath("jre", "17", "extracted", "bin", Sys.javaExecutable())

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

	procAttr := &os.ProcAttr{
		Dir:   alpinePath(),
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	}

	sentry.Breadcrumb(ctx, fmt.Sprintf("starting launcher process: %s %s", jrePath, args))
	proc, err := os.StartProcess(jrePath, args, procAttr)
	if err != nil {
		sentry.Breadcrumb(ctx, fmt.Sprintf("failed to start launcher (retrying): %v", err))
		// retry with regular java.exe
		proc, err = os.StartProcess(alpinePath("jre", "17", "extracted", "bin", "java"), args, procAttr)
		if err != nil {
			return TaskResult{ctx, err}
		}
	}

	sentry.Breadcrumb(ctx, "releasing launcher process")
	err = proc.Release()
	if err != nil {
		return TaskResult{ctx, err}
	}

	return TaskResult{ctx, nil}
}

func cleanup() {
	_ = os.RemoveAll(alpinePath("launcher.jar"))
	_ = os.RemoveAll(alpinePath("jre", "17"))
}
