package main

import (
	"os"
)

func (sys OperatingSystem) javaExecutable() string {
	if sys == Windows {
		return "javaw.exe"
	}
	return "java"
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
