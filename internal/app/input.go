package app

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"
)

func ReadSecret(prompt string) (string, error) {
	fmt.Print(prompt)
	pwBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	log.Println()
	if err != nil {
		log.Println("Error reading passphrase:", err)
		return "", err
	}
	return string(pwBytes), nil
}

func ReadPassword(isNew ...bool) (string, error) {
	password := ""

	prompt := "Enter password: "
	if len(isNew) > 0 && isNew[0] {
		prompt = "Enter new password: "
	}

	for {
		fmt.Print(prompt)
		pwBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			log.Println()
			log.Println("Error reading password:", err)
			return "", err
		}
		password = string(pwBytes)
		log.Println()

		fmt.Print("Confirm password: ")
		confirmBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			log.Println()
			log.Println("Error reading password:", err)
			return "", err
		}
		log.Println()
		if password == string(confirmBytes) {
			break
		}
		log.Println("Password did not match, try again")
		log.Println()
	}

	return password, nil
}

func ReadCredential() *Credentials {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Println("Error getting home directory:", err)
		return nil
	}
	passwordsDir := filepath.Join(home, ".local", "share", "vd", "passwords")

	emailPath := filepath.Join(home, ".local", "share", "vd", "gpg_email.txt")
	emailBytes, err := os.ReadFile(emailPath)
	if err != nil || len(strings.TrimSpace(string(emailBytes))) == 0 {
		log.Println("No GPG key registered. Run `vd register` first.")
		return nil
	}

	var name string

	fmt.Print("Enter name: ")
	fmt.Scanln(&name)

	filename := filepath.Join(passwordsDir, name)
	if _, err := os.Stat(filename); err == nil {
		log.Printf("Error: password for %s already exists", name)
		return nil
	}

	password, err := ReadPassword()
	if err != nil {
		return nil
	}

	return &Credentials{Name: name, Password: password}
}
