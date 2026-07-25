package app

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/term"
)

func register() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Println("Error getting home directory:", err)
		return
	}

	vdDir := filepath.Join(home, ".local", "share", "vd")

	if err := os.MkdirAll(vdDir, 0o700); err != nil {
		log.Println("Error creating vd directory:", err)
		return
	}

	emailPath := filepath.Join(vdDir, "gpg_email.txt")

	if _, err := os.Stat(emailPath); err == nil {
		log.Printf("Error: password for %s already exists", emailPath)
		return
	}

	var name, email, password string

	fmt.Print("Enter name: ")
	fmt.Scanln(&name)
	fmt.Print("Enter email: ")
	fmt.Scanln(&email)
	for {
		fmt.Print("Enter master password: ")
		pwBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			log.Println()
			log.Println("Error reading password:", err)
			return
		}
		password = string(pwBytes)
		log.Println()

		fmt.Print("Confirm master password: ")
		confirmBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			log.Println()
			log.Println("Error reading password:", err)
			return
		}
		log.Println()
		if password == string(confirmBytes) {
			break
		}
		log.Println("Master password did not match, try again")
		log.Println()
	}

	if err := GenerateGPGKey(name, email, password); err != nil {
		log.Println("Error generating GPG key:", err)
		return
	}

	if err := os.WriteFile(emailPath, []byte(email), 0o600); err != nil {
		log.Println("Error storing email:", err)
		return
	}
	log.Println("Registration complete")
}
