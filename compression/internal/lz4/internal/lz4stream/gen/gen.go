//go:build ignore

package main

import "log"

//go:generate go run gen.go

func main() {
	if err := do(); err != nil {
		log.Fatal(err)
	}
}
