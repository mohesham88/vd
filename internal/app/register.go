package app

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/term"
)

func RegisteredEmailPath() (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}

	emailPath := filepath.Join(home, ".local", "share", "vd", "gpg_email.txt")
	if _, err := os.Stat(emailPath); err != nil {
		return "", false
	}

	return emailPath, true
}

func Register(name, email, password string) error {
	if path, ok := RegisteredEmailPath(); ok {
		return fmt.Errorf("you already registered with %s", path)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error getting home directory: %v", err)
	}

	vdDir := filepath.Join(home, ".local", "share", "vd")

	if err := os.MkdirAll(vdDir, 0o700); err != nil {
		return fmt.Errorf("error creating vd directory: %v", err)
	}

	emailPath := filepath.Join(vdDir, "gpg_email.txt")

	if err := GenerateGPGKey(name, email, password); err != nil {
		return fmt.Errorf("error generating GPG key: %v", err)
	}

	if err := os.WriteFile(emailPath, []byte(email), 0o600); err != nil {
		return fmt.Errorf("error storing email: %v", err)
	}

	return nil
}

func register() {
	if path, ok := RegisteredEmailPath(); ok {
		fmt.Printf("Error: You already registered with %s\n", path)
		return
	}

	var password string

	name, err := ScanLine("Enter name: ")
	if err != nil {
		fmt.Println("Error reading name:", err)
		return
	}
	email, err := ScanLine("Enter email: ")
	if err != nil {
		fmt.Println("Error reading email:", err)
		return
	}
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

	if err := Register(name, email, password); err != nil {
		log.Println(err)
		return
	}

	log.Println("Registration complete")
}
