package main

import "golang.org/x/tools/cmd/stringer"

//go:generate go run golang.org/x/tools/cmd/stringer -type=BlockSize,CompressionLevel -output ../options_gen.go

func main() {

}
