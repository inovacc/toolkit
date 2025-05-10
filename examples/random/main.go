package main

import (
	"fmt"

	"github.com/inovacc/utils/v2/random/password"
	"github.com/inovacc/utils/v2/random/random"
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
