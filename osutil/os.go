package osutil

import (
	"os"
	"runtime"
)

func SelectOS() string {
	switch runtime.GOOS {
	case "windows":
		return "windows"
	case "darwin":
		return "macos"
	default:
		return "linux"
	}
}

func SelectArch() string {
	switch runtime.GOARCH {
	case "386":
		return "i386"
	case "arm64":
		return "aarch64"
	default:
		return "x86_64"
	}
}

func HomeDirFromEnv() string {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		homeDir = os.Getenv("USERPROFILE")
	}
	return homeDir
}
