//go:build linux

package osutil

import "testing"

func TestSelectOS(t *testing.T) {
	name := SelectOS()
	if name == "" {
		t.Error("SelectOS() = empty")
	}
}

func TestSelectArch(t *testing.T) {
	name := SelectArch()
	if name == "" {
		t.Error("SelectArch() = empty")
	}
}

func TestHomeDirFromEnv(t *testing.T) {
	name := HomeDirFromEnv()
	if name == "" {
		t.Error("HomeDirFromEnv() = empty")
	}
}
