package osutil

import (
	"os"
	"runtime"
)

func SelectOS() string {
	o := "linux"
	if runtime.GOOS == "windows" {
		o = "windows"
	} else if runtime.GOOS == "darwin" {
		o = "macos"
	}
	return o
}

func SelectArch() string {
	arch := "x86_64"
	if runtime.GOARCH == "386" {
		arch = "i386"
	} else if runtime.GOARCH == "arm64" {
		arch = "aarch64"
	}
	return arch
}

func HomeDirFromEnv() string {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		homeDir = os.Getenv("USERPROFILE")
	}
	return homeDir
}
