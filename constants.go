package main

type (
	OperatingSystem string
	Architecture    string
)

const (
	Windows OperatingSystem = "windows"
	Linux   OperatingSystem = "linux"
	Mac     OperatingSystem = "darwin"
)

const (
	x86   Architecture = "amd64"
	Arm64 Architecture = "arm64"
)

const (
	MetadataURL      string = "https://metadata.alpineclient.com"
	GitHubReleaseURL string = "https://api.github.com/repos/alpine-client/pinnacle/releases/latest"
	SupportEmail     string = "contact@crystaldev.co"
)
