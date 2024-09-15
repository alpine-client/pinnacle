package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/alpine-client/pinnacle/sentry"
	"github.com/alpine-client/pinnacle/ui"
)

type ArchiveEntry interface {
	Name() string
	IsDir() bool
	IsSymlink() bool
	Mode() (os.FileMode, error)
	Size() (int64, error)
	Open() (io.ReadCloser, error)
	Linkname() (string, error)
}

type Archive interface {
	Next() (ArchiveEntry, error)
	Close() error
	TotalSize() int64
}

// TarEntry implements ArchiveEntry for tar files.
type TarEntry struct {
	header *tar.Header
	reader *tar.Reader
}

func (te *TarEntry) Name() string {
	return te.header.Name
}

func (te *TarEntry) IsDir() bool {
	return te.header.Typeflag == tar.TypeDir
}

func (te *TarEntry) IsSymlink() bool {
	return te.header.Typeflag == tar.TypeSymlink
}

func (te *TarEntry) Mode() (os.FileMode, error) {
	mode, err := safeInt64ToUint32(te.header.Mode)
	if err != nil {
		return 0, err
	}
	return os.FileMode(mode), nil
}

func (te *TarEntry) Size() (int64, error) {
	return te.header.Size, nil
}

func (te *TarEntry) Open() (io.ReadCloser, error) {
	return io.NopCloser(io.LimitReader(te.reader, te.header.Size)), nil
}

func (te *TarEntry) Linkname() (string, error) {
	return te.header.Linkname, nil
}

// TarArchive implements Archive for tar files.
type TarArchive struct {
	reader     *tar.Reader
	gzipReader *gzip.Reader
	file       *os.File
	totalSize  int64
}

func NewTarArchive(file *os.File) (Archive, error) {
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	tarReader := tar.NewReader(gzReader)

	// Calculate total size
	var totalSize int64
	var header *tar.Header
	for {
		header, err = tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		totalSize += header.Size
	}

	// Reset readers to start from the beginning again
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}
	err = gzReader.Reset(file)
	if err != nil {
		return nil, err
	}
	tarReader = tar.NewReader(gzReader)

	return &TarArchive{
		reader:     tarReader,
		gzipReader: gzReader,
		file:       file,
		totalSize:  totalSize,
	}, nil
}

func (ta *TarArchive) Next() (ArchiveEntry, error) {
	header, err := ta.reader.Next()
	if err != nil {
		return nil, err
	}
	return &TarEntry{header: header, reader: ta.reader}, nil
}

func (ta *TarArchive) Close() error {
	err1 := ta.gzipReader.Close()
	err2 := ta.file.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

func (ta *TarArchive) TotalSize() int64 {
	return ta.totalSize
}

// ZipEntry implements ArchiveEntry for zip files.
type ZipEntry struct {
	file *zip.File
}

func (ze *ZipEntry) Name() string {
	return ze.file.Name
}

func (ze *ZipEntry) IsDir() bool {
	return ze.file.FileInfo().IsDir()
}

func (ze *ZipEntry) IsSymlink() bool {
	return ze.file.Mode()&os.ModeSymlink != 0
}

func (ze *ZipEntry) Mode() (os.FileMode, error) {
	return ze.file.Mode(), nil
}

func (ze *ZipEntry) Size() (int64, error) {
	return safeUint64ToInt64(ze.file.UncompressedSize64)
}

func (ze *ZipEntry) Open() (io.ReadCloser, error) {
	return ze.file.Open()
}

func (ze *ZipEntry) Linkname() (string, error) {
	if !ze.IsSymlink() {
		return "", nil
	}
	rc, err := ze.file.Open()
	if err != nil {
		return "", err
	}
	err = rc.Close()
	if err != nil {
		return "", err
	}
	linkname, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}
	return string(linkname), nil
}

// ZipArchive implements Archive for zip files.
type ZipArchive struct {
	reader    *zip.Reader
	totalSize int64
	index     int
}

func NewZipArchive(file *os.File) (Archive, error) {
	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}
	reader, err := zip.NewReader(file, fi.Size())
	if err != nil {
		return nil, err
	}

	// Calculate total size
	var totalSize int64
	var ts int64
	for _, f := range reader.File {
		ts, err = safeUint64ToInt64(f.UncompressedSize64)
		if err != nil {
			return nil, err
		}
		totalSize += ts
	}

	return &ZipArchive{
		reader:    reader,
		totalSize: totalSize,
		index:     0,
	}, nil
}

func (za *ZipArchive) Next() (ArchiveEntry, error) {
	if za.index >= len(za.reader.File) {
		return nil, io.EOF
	}
	entry := &ZipEntry{file: za.reader.File[za.index]}
	za.index++
	return entry, nil
}

func (za *ZipArchive) Close() error {
	return nil
}

func (za *ZipArchive) TotalSize() int64 {
	return za.totalSize
}

// detectFormat detects the archive format based on the file content.
func detectFormat(file *os.File) (ArchiveType, error) {
	buf := make([]byte, 4)
	_, err := file.Read(buf)
	if err != nil {
		return "", err
	}
	switch {
	case buf[0] == 0x50 && buf[1] == 0x4b: // ZIP header
		return Zip, nil
	case buf[0] == 0x1f && buf[1] == 0x8b: // GZIP header
		return TarGz, nil
	default:
		return "", errors.New("unknown archive format")
	}
}

func extractFile(ctx context.Context, entry ArchiveEntry, target string) error {
	err := os.MkdirAll(filepath.Dir(target), os.ModePerm)
	if err != nil {
		return err
	}

	fileMode, err := entry.Mode()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileMode)
	if err != nil {
		return err
	}
	defer func() {
		sentry.CaptureErr(ctx, out.Close())
	}()

	rc, err := entry.Open()
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

func extractArchive(ctx context.Context, src string, dest string, pt *ui.ProgressiveTask) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		sentry.CaptureErr(ctx, file.Close())
	}()

	format, err := detectFormat(file)
	if err != nil {
		return err
	}

	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	var archive Archive
	switch format {
	case Zip:
		archive, err = NewZipArchive(file)
		if err != nil {
			return err
		}
	case TarGz:
		archive, err = NewTarArchive(file)
		if err != nil {
			return err
		}
	default:
		return errors.New("unsupported archive format")
	}
	defer func() {
		sentry.CaptureErr(ctx, archive.Close())
	}()

	err = os.MkdirAll(dest, os.ModePerm)
	if err != nil {
		return err
	}

	var bytesProcessed int64
	var entry ArchiveEntry
	totalSize := archive.TotalSize()
	for {
		entry, err = archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}

		parts := strings.Split(entry.Name(), "/")
		if len(parts) > 1 {
			parts = parts[1:] // strip components
		}
		target := filepath.Join(dest, filepath.Join(parts...))

		switch {
		case entry.IsSymlink():
			var linkname string
			linkname, err = entry.Linkname()
			if err != nil {
				return err
			}
			err = os.Symlink(linkname, target)
			if err != nil {
				return err
			}
		case entry.IsDir():
			var fileMode os.FileMode
			fileMode, err = entry.Mode()
			if err != nil {
				return err
			}
			err = os.MkdirAll(target, fileMode)
			if err != nil {
				return err
			}
		default:
			err = extractFile(ctx, entry, target)
			if err != nil {
				return err
			}
		}

		var b int64
		b, err = entry.Size()
		if err != nil {
			return err
		}
		bytesProcessed += b
		if pt != nil {
			pt.UpdateProgress(float64(bytesProcessed) / float64(totalSize))
		}
	}

	return nil
}
