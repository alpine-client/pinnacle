package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
)

func systemInformation() (OperatingSystem, Architecture) {
	var sys OperatingSystem
	var arch Architecture

	switch runtime.GOOS {
	case "windows":
		sys = Windows
	case "linux":
		sys = Linux
	case "darwin":
		sys = Mac
	default:
		panic("unsupported operating system")
	}

	switch runtime.GOARCH {
	case "amd64":
		arch = x86
	case "arm64":
		arch = Arm64
	default:
		panic("unsupported system architecture")
	}

	return sys, arch
}

func archiveFormat(os OperatingSystem, arch Architecture) ArchiveType {
	if os == Linux && arch == Arm64 {
		return TarGz
	}
	return Zip
}

func (sys OperatingSystem) javaExecutable() string {
	if sys == Windows {
		return "javaw.exe"
	}
	return "java"
}

// alpinePath returns the absolute path of Alpine Client's
// data directory based on the user's operating system.
//
// Optionally, pass in sub-folder/file names to add
// them to the returned path.
// - Example: alpinePath("jre", "17", "version.json")
//
// Windows - %AppData%\.alpineclient
// Mac - $HOME/Library/Application Support/alpineclient
// Linux - $HOME/.alpineclient
//
// note: The missing '.' for macOS is intentional.
func alpinePath(subs ...string) string {
	var baseDir string
	var dirs []string

	switch Sys {
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func safeUint64ToInt64(val uint64) (int64, error) {
	if val > math.MaxInt64 {
		return 0, fmt.Errorf("integer overflow: cannot convert %d to int64", val)
	}
	return int64(val), nil
}

func safeInt64ToUint32(val int64) (uint32, error) {
	if val < 0 || val > math.MaxUint32 {
		return 0, fmt.Errorf("integer overflow: cannot convert %d to uint32", val)
	}
	return uint32(val), nil
}
