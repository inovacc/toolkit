package main

import (
	"fmt"

	"github.com/inovacc/toolkit/data/algorithm/random"
	"github.com/inovacc/toolkit/data/password"
)

func main() {
	// generate strong password
	randPass := password.NewPassword(
		password.WithLength(16),
		password.WithNumbers(),
		password.WithSpecial(),
		password.WithLower(),
		password.WithUpper(),
	)
	pass, _ := randPass.Generate()
	fmt.Println("Strong Password:", pass)

	randNum, _ := random.RandomInt(1, 100)
	fmt.Println("Random Number:", randNum)
}
