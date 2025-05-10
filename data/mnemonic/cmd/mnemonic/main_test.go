package main

import (
	"os"
	"testing"
)

func TestMain_CLI(t *testing.T) {
	os.Args = []string{"cmd", "-n", "1", "-lang=English", "-f=sentence"}
	main()
}

func TestMain_Template(t *testing.T) {
	os.Args = []string{"cmd", "-n", "1", "-f=template", "-t=mnemonic: {{.Sentence}}"}
	main()
}
