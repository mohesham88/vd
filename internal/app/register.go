package app

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/term"
)

func register() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory:", err)
		return
	}

	vdDir := filepath.Join(home, ".local", "share", "vd")

	if err := os.MkdirAll(vdDir, 0o700); err != nil {
		fmt.Println("Error creating vd directory:", err)
		return
	}

	emailPath := filepath.Join(vdDir, "gpg_email.txt")

	if _, err := os.Stat(emailPath); err == nil {
		fmt.Printf("Error: password for %s already exists\n", emailPath)
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
			fmt.Println()
			fmt.Println("Error reading password:", err)
			return
		}
		password = string(pwBytes)
		fmt.Println()

		fmt.Print("Confirm master password: ")
		confirmBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			fmt.Println()
			fmt.Println("Error reading password:", err)
			return
		}
		fmt.Println()
		if password == string(confirmBytes) {
			break
		}
		fmt.Println("Master password did not match, try again")
		fmt.Println()
	}

	if err := GenerateGPGKey(name, email, password); err != nil {
		fmt.Println("Error generating GPG key:", err)
		return
	}

	if err := os.WriteFile(emailPath, []byte(email), 0o600); err != nil {
		fmt.Println("Error storing email:", err)
		return
	}
	fmt.Println("Registration complete")
}
