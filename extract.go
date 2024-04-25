package main

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/alpine-client/pinnacle/sentry"
	"github.com/alpine-client/pinnacle/ui"
)

func isSymLink(file *zip.File) bool {
	return file.Mode()&os.ModeSymlink != 0
}

func extractSymLink(ctx context.Context, file *zip.File, target string) error {
	var rc io.ReadCloser
	var out []byte
	var err error

	rc, err = file.Open()
	if err != nil {
		return err
	}
	defer func() {
		sentry.CaptureErr(ctx, rc.Close())
	}()

	out, err = io.ReadAll(rc)
	if err != nil {
		return err
	}

	err = os.Symlink(string(out), target)
	if err != nil {
		return err
	}
	fmt.Printf("created symlink %s", target)
	return nil
}

func extractFile(ctx context.Context, file *zip.File, target string) error {
	var rc io.ReadCloser
	var out *os.File
	var err error

	err = os.MkdirAll(filepath.Dir(target), os.ModePerm)
	if err != nil {
		return err
	}

	out, err = os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return err
	}
	defer func() {
		sentry.CaptureErr(ctx, out.Close())
	}()

	rc, err = file.Open()
	if err != nil {
		return err
	}
	defer func() {
		sentry.CaptureErr(ctx, rc.Close())
	}()

	_, err = io.Copy(out, rc)
	if err != nil {
		return err
	}

	return nil
}

func extractAll(ctx context.Context, src string, dest string) error {
	zipReader, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		sentry.CaptureErr(ctx, zipReader.Close())
	}()

	err = os.MkdirAll(dest, os.ModePerm)
	if err != nil {
		return err
	}

	var progress int
	total := len(zipReader.File)
	symlinks := make(map[string]*zip.File, len(zipReader.File))
	for _, file := range zipReader.File {
		parts := strings.Split(file.Name, "/")
		if len(parts) > 1 {
			parts = parts[1:] // strip components
		}
		target := filepath.Join(dest, filepath.Join(parts...))

		if isSymLink(file) {
			symlinks[target] = file
			continue
		}

		progress++
		ui.UpdateProgress(1, fmt.Sprintf("Extracting java (%d/%d)...", progress, total))

		if file.FileInfo().IsDir() {
			err = os.MkdirAll(target, os.ModePerm)
			if err != nil {
				return err
			}
			continue
		}

		err = extractFile(ctx, file, target)
		if err != nil {
			return err
		}
	}

	for path, link := range symlinks {
		progress++
		ui.UpdateProgress(1, fmt.Sprintf("Extracting java (%d/%d)...", progress, total))
		err = extractSymLink(ctx, link, path)
		if err != nil {
			return err
		}
	}

	return nil
}
