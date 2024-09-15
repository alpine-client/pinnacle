package main

type (
	OperatingSystem string
	Architecture    string
	ArchiveType     string
)

const (
	Windows OperatingSystem = "windows"
	Linux   OperatingSystem = "linux"
	Mac     OperatingSystem = "macos"
)

const (
	x86   Architecture = "x86"
	Arm64 Architecture = "arm"
)

const (
	Zip   ArchiveType = "zip"
	TarGz ArchiveType = "tar.gz"
)

const (
	MetadataURL      string = "https://metadata.alpineclient.com"
	GitHubReleaseURL string = "https://api.github.com/repos/alpine-client/pinnacle/releases/latest"
	SupportEmail     string = "contact@crystaldev.co"
)
