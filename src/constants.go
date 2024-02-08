package main

type OperatingSystem string
type Architecture string

const (
	MetadataURL  string = "https://metadata.alpineclientprod.com"
	SupportURL   string = "https://discord.alpineclient.com"
	SupportEmail string = "contact@crystaldev.co"
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
