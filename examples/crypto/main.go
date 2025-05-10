package main

import (
	"fmt"

	"github.com/inovacc/toolkit/data/algorithm/hashing"
	"github.com/inovacc/toolkit/data/password"
)

func main() {
	newHasher := hashing.NewHasher(hashing.SHA256)
	fmt.Println("SHA256 hash:", newHasher.HashString("hello world"))

	passBcrypt, _ := password.HashPasswordBcrypt("secret")
	fmt.Println("Bcrypt Password hash:", passBcrypt)

	passArgon2, _ := password.HashPasswordArgon2("secret", nil)
	fmt.Println("Argon2 Password hash:", passArgon2)
}
