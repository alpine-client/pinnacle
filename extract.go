package main

import (
	"archive/zip"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/alpine-client/pinnacle/ui"
)

func isSymLink(file *zip.File) bool {
	return file.Mode()&os.ModeSymlink != 0
}

func (p *Pinnacle) extractArchive(ctx context.Context, src string, dest string, pt *ui.ProgressiveTask) error {
	p.Breadcrumb(ctx, "extracting archive "+src+" to "+dest)
	err := os.MkdirAll(dest, os.ModePerm)
	if err != nil {
		return err
	}
	if p.os == Linux && p.arch == Arm64 {
		return p.extractTar(src, dest)
	}
	return p.extractZip(ctx, src, dest, pt)
}

func (p *Pinnacle) extractSymLink(ctx context.Context, file *zip.File, target string) error {
	var rc io.ReadCloser
	var out []byte
	var err error

	rc, err = file.Open()
	if err != nil {
		return err
	}
	defer func() {
		p.CaptureErr(ctx, rc.Close())
	}()

	out, err = io.ReadAll(rc)
	if err != nil {
		return err
	}

	err = os.Symlink(string(out), target)
	if err != nil {
		return err
	}

	return nil
}

func (p *Pinnacle) extractFile(ctx context.Context, file *zip.File, target string) error {
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
		p.CaptureErr(ctx, out.Close())
	}()

	rc, err = file.Open()
	if err != nil {
		return err
	}
	defer func() {
		p.CaptureErr(ctx, rc.Close())
	}()

	_, err = io.Copy(out, rc)
	if err != nil {
		return err
	}

	return nil
}

func (p *Pinnacle) extractZip(ctx context.Context, src string, dest string, pt *ui.ProgressiveTask) error {
	zipReader, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		p.CaptureErr(ctx, zipReader.Close())
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
		if pt != nil {
			pt.UpdateProgress(float64(progress) / float64(total))
		}

		if file.FileInfo().IsDir() {
			err = os.MkdirAll(target, os.ModePerm)
			if err != nil {
				return err
			}
			continue
		}

		err = p.extractFile(ctx, file, target)
		if err != nil {
			return err
		}
	}

	for path, link := range symlinks {
		progress++
		if pt != nil {
			pt.UpdateProgress(float64(progress) / float64(total))
		}
		err = p.extractSymLink(ctx, link, path)
		if err != nil {
			return err
		}
	}

	return nil
}

func (*Pinnacle) extractTar(src string, dest string) error {
	cmd := exec.Command("tar", "--strip-components=1", "-xzf", src, "-C", dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
