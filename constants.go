package main

type (
	OperatingSystem string
	Architecture    string
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
	MetadataURL      string = "https://metadata.alpineclientprod.com"
	GitHubReleaseURL string = "https://api.github.com/repos/alpine-client/pinnacle/releases/latest"
	SupportEmail     string = "contact@crystaldev.co"
)
