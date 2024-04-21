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
	MetadataURL  string = "https://metadata.alpineclientprod.com"
	SupportEmail string = "contact@crystaldev.co"
)
