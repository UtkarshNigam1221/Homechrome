//go:build ignore
// +build ignore

// Script to generate bcrypt password hashes for development
// Usage: go run scripts/generate-password-hash.go <password>

package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run scripts/generate-password-hash.go <password>")
		fmt.Println("Example: go run scripts/generate-password-hash.go 'Admin@123!'")
		os.Exit(1)
	}

	password := os.Args[1]

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		fmt.Printf("Error generating hash: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Password: %s\n", password)
	fmt.Printf("Hash: %s\n", string(hash))
}
