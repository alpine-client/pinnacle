package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/alpine-client/pinnacle/sentry"
	"github.com/alpine-client/pinnacle/ui"
)

type Pinnacle struct {
	os      OperatingSystem
	arch    Architecture
	logger  *slog.Logger
	logFile *os.File
	version string
	branch  string
}

type MetadataResponse struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Hash string `json:"sha1"`
	Size uint32 `json:"size"`
}

type JavaManifest struct {
	Hash string `json:"checksum"`
	Size uint32 `json:"size"`
}

var (
	metadataResponse MetadataResponse

	errMissingJava     = errors.New("missing java")
	errMissingLauncher = errors.New("missing launcher")
)

func (p *Pinnacle) setup() error {
	var err error

	// Set Launcher Branch
	branch := flag.String("branch", "production", "Launcher branch")
	flag.Parse()
	p.branch = *branch

	// Setup Logger
	err = os.MkdirAll(p.alpinePath("logs"), os.ModePerm) // note: creates .alpineclient AND .alpineclient/logs
	if err != nil {
		return err
	}

	p.logFile, err = os.OpenFile(p.alpinePath("logs", "updater.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o666)
	if err != nil {
		return err
	}

	p.logger = slog.New(slog.NewTextHandler(io.MultiWriter(os.Stdout, os.Stderr, p.logFile), nil))
	log.SetOutput(io.MultiWriter(os.Stdout, os.Stderr, p.logFile))

	// Setup Sentry
	p.StartSentry(version, p.fetchSentryDSN())

	return nil
}

func (p *Pinnacle) cleanup(ctx context.Context, err error) {
	if err == nil {
		return
	}
	p.CaptureErr(ctx, ui.DisplayError(ctx, err, p.logFile))
	p.CaptureErr(ctx, os.RemoveAll(p.alpinePath("launcher.jar")))
	p.CaptureErr(ctx, os.RemoveAll(p.alpinePath("jre", "17")))
}

func (p *Pinnacle) download(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, errMissingLauncher):
		ui.Render()
		return p.downloadLauncher(ctx)
	case errors.Is(err, errMissingJava):
		ui.Render()
		return p.downloadJava(ctx)
	}
	return err
}

func (p *Pinnacle) error(_ context.Context, err error) error {
	return err
}

func (p *Pinnacle) fetchMetadata(ctx context.Context, url string) (*MetadataResponse, error) {
	resp, err := p.getFromURL(ctx, url)
	if err != nil {
		return nil, err
	}

	defer func() {
		p.CaptureErr(ctx, resp.Body.Close())
	}()

	p.Breadcrumb(ctx, "decoding response from "+url)

	err = json.NewDecoder(resp.Body).Decode(&metadataResponse)
	if err != nil {
		return nil, err
	}

	return &metadataResponse, nil
}

func (p *Pinnacle) fileHashMatches(ctx context.Context, hash string, path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer func() {
		p.CaptureErr(ctx, file.Close())
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

func (p *Pinnacle) checkLauncher(ctx context.Context) error {
	pt := ui.NewProgressTask("Preparing launcher...")

	p.Breadcrumb(ctx, "fetching metadata from /pinnacle")
	pt.UpdateProgress(0.20)

	launcher, err := p.fetchMetadata(ctx, MetadataURL+"/pinnacle?branch="+p.branch)
	if err != nil {
		return err
	}
	pt.UpdateProgress(0.60)

	targetPath := p.alpinePath("launcher.jar")
	if !fileExists(targetPath) {
		p.Breadcrumb(ctx, "missing launcher.jar")
		return errMissingLauncher
	}

	if validHash, _ := p.fileHashMatches(ctx, launcher.Hash, targetPath); !validHash {
		p.Breadcrumb(ctx, "failed checksum validation")
		return errMissingLauncher
	}

	p.Breadcrumb(ctx, "finished checkLauncher (jar existed)")
	return nil
}

func (p *Pinnacle) downloadLauncher(ctx context.Context) error {
	pt := ui.NewProgressTask("Downloading launcher...")
	dest := p.alpinePath("launcher.jar")

	err := p.downloadFile(ctx, metadataResponse.URL, dest, pt)
	if err != nil {
		return err
	}

	var validHash bool
	if validHash, err = p.fileHashMatches(ctx, metadataResponse.Hash, dest); !validHash {
		p.Breadcrumb(ctx, fmt.Sprintf("hash mismatch after download (retry): %v", err), slog.LevelError)

		_ = os.RemoveAll(dest)
		err = p.downloadFile(ctx, metadataResponse.URL, dest, pt)
		if err != nil {
			return err
		}

		if validHash, err = p.fileHashMatches(ctx, metadataResponse.Hash, dest); !validHash {
			p.Breadcrumb(ctx, fmt.Sprintf("hash mismatch after download: %v", err), slog.LevelError)
			return err
		}
	}

	p.Breadcrumb(ctx, "finished checkLauncher (jar downloaded)")
	pt.UpdateProgress(0.99999, "Starting launcher...")
	return nil
}

func (p *Pinnacle) checkJava(ctx context.Context) error {
	pt := ui.NewProgressTask("Preparing Java runtime...")
	path := p.alpinePath("jre", "17")

	p.Breadcrumb(ctx, "mkdir "+path)
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}
	pt.UpdateProgress(0.20)

	endpoint := fmt.Sprintf("%s/jre?version=17&os=%s&arch=%s", MetadataURL, p.os, p.arch)
	p.Breadcrumb(ctx, "fetching manifest from "+endpoint)
	jre, err := p.fetchMetadata(ctx, endpoint)
	if err != nil {
		return err
	}

	var data []byte
	var manifest JavaManifest
	javaPath := p.alpinePath("jre", "17", "extracted", "bin", p.os.javaExecutable())
	manifestPath := p.alpinePath("jre", "17", "version.json")
	pt.UpdateProgress(0.50)

	if !fileExists(javaPath) {
		p.Breadcrumb(ctx, "missing java executable")
		return errMissingJava
	}
	pt.UpdateProgress(0.68)

	if !fileExists(manifestPath) {
		p.Breadcrumb(ctx, "missing manifest")
		return errMissingJava
	}
	pt.UpdateProgress(0.76)

	data, err = os.ReadFile(manifestPath)
	if err != nil {
		p.Breadcrumb(ctx, "failed to read manifest file")
		return errMissingJava
	}
	pt.UpdateProgress(0.85)

	if err = json.Unmarshal(data, &manifest); err != nil {
		p.Breadcrumb(ctx, "failed to unmarshal manifest file")
		return errMissingJava
	}
	pt.UpdateProgress(0.98)

	if manifest.Hash != jre.Hash {
		p.Breadcrumb(ctx, fmt.Sprintf("file checksum  %s does not match expected %s", manifest.Hash, jre.Hash))
		return errMissingJava
	}

	p.Breadcrumb(ctx, "finished checkJava (existed)")
	return nil
}

func (p *Pinnacle) downloadJava(ctx context.Context) error {
	archivePath := p.alpinePath("jre", "17", metadataResponse.Name)
	extractedPath := p.alpinePath("jre", "17", "extracted")
	manifestPath := p.alpinePath("jre", "17", "version.json")

	p.CaptureErr(ctx, os.RemoveAll(archivePath))
	p.CaptureErr(ctx, os.RemoveAll(extractedPath))
	p.CaptureErr(ctx, os.RemoveAll(manifestPath))

	pt := ui.NewProgressTask("Downloading Java...")
	err := p.downloadFile(ctx, metadataResponse.URL, archivePath, pt)
	if err != nil {
		return err
	}

	pt = ui.NewProgressTask("Extracting Java...")

	err = p.extractArchive(ctx, archivePath, extractedPath, pt)
	if err != nil {
		return err
	}

	_ = os.Chmod(p.alpinePath("jre", "17", "extracted", "bin", p.os.javaExecutable()), 0o755)

	data, err := json.Marshal(JavaManifest{Hash: metadataResponse.Hash, Size: metadataResponse.Size})
	if err != nil {
		return err
	}

	p.Breadcrumb(ctx, "writing manifest to file "+manifestPath)
	err = os.WriteFile(manifestPath, data, 0o600)
	if err != nil {
		return err
	}

	_ = os.Remove(archivePath)
	p.Breadcrumb(ctx, "finished checkJava (downloaded)")
	return nil
}

func (p *Pinnacle) startLauncher(ctx context.Context) error {
	pt := ui.NewProgressTask("Starting launcher...")
	pt.UpdateProgress(0.50)

	jarPath := p.alpinePath("launcher.jar")
	jrePath := p.alpinePath("jre", "17", "extracted", "bin", p.os.javaExecutable())

	args := []string{
		"-Xms256M",
		"-Xmx256M",
	}

	if p.os == Mac {
		args = append(args, "-XstartOnFirstThread")
	}

	args = append(args, "-jar", jarPath)

	if version != "" {
		args = append(args, "--pinnacle-version", version)
	}

	procAttr := &os.ProcAttr{
		Dir:   p.alpinePath(),
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	}

	pt.UpdateProgress(0.75)
	p.Breadcrumb(ctx, fmt.Sprintf("starting launcher process: %s %s", jrePath, args))
	proc, err := os.StartProcess(jrePath, args, procAttr)
	if err != nil {
		p.Breadcrumb(ctx, fmt.Sprintf("failed to start launcher (retrying): %v", err))
		// retry with regular java.exe
		proc, err = os.StartProcess(p.alpinePath("jre", "17", "extracted", "bin", "java"), args, procAttr)
		if err != nil {
			return err
		}
	}

	pt.UpdateProgress(0.95)
	p.Breadcrumb(ctx, "releasing launcher process")
	err = proc.Release()
	if err != nil {
		return err
	}

	pt.UpdateProgress(0.99)
	return nil
}

func (p *Pinnacle) StartSentry(release string, dsn string) {
	err := sentry.Start(release, dsn)
	if err != nil {
		p.logger.Warn(err.Error())
	}
}

func (p *Pinnacle) CaptureErr(ctx context.Context, err error) {
	if err == nil {
		return
	}
	p.logger.ErrorContext(ctx, err.Error())
	sentry.CaptureErr(ctx, err)
}

func (p *Pinnacle) Breadcrumb(ctx context.Context, desc string, level ...slog.Level) {
	var lvl slog.Level
	if len(level) == 0 {
		lvl = slog.LevelInfo
	} else {
		lvl = level[0]
	}
	p.logger.Log(ctx, lvl, desc)
	sentry.Breadcrumb(ctx, desc, lvl)
}

// alpinePath returns the absolute path of Alpine Client's
// data directory based on the user's operating system.
//
// Optionally, pass in sub-folder/file names to add
// them to the returned path.
// - Example: p.alpinePath("jre", "17", "version.json")
//
// Windows - %AppData%\.alpineclient
// Mac - $HOME/Library/Application Support/alpineclient
// Linux - $HOME/.alpineclient
//
// note: The missing '.' for macOS is intentional.
func (p *Pinnacle) alpinePath(subs ...string) string {
	var baseDir string
	var dirs []string

	switch p.os {
	case Windows:
		baseDir = os.Getenv("AppData")
		dirs = append(dirs, baseDir, ".alpineclient")
	case Mac:
		baseDir = os.Getenv("HOME")
		dirs = append(dirs, baseDir, "Library", "Application Support", "alpineclient")
	case Linux:
		baseDir = os.Getenv("HOME")
		dirs = append(dirs, baseDir, ".alpineclient")
	}

	return filepath.Join(append(dirs, subs...)...)
}
