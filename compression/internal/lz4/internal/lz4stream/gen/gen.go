//go:build ignore

package main

import "log"

func main() {
	if err := do(); err != nil {
		log.Fatal(err)
	}
}
