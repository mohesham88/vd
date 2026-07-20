package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"
)

func readMasterPassword() string {
	fmt.Print("Enter master password: ")
	pwBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Println()
		fmt.Println("Error reading master password:", err)
		return ""
	}
	fmt.Println()
	return string(pwBytes)
}

func readCredential() map[string]string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory:", err)
		return nil
	}
	dir := filepath.Join(home, ".local", "share", "vd", "passwords")

	emailPath := filepath.Join(home, ".local", "share", "vd", "gpg_email.txt")
	emailBytes, err := os.ReadFile(emailPath)
	if err != nil || len(strings.TrimSpace(string(emailBytes))) == 0 {
		fmt.Println("No GPG key registered. Run `vd register` first.")
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		fmt.Println("Error reading passwords directory:", err)
		return nil
	}

	if len(entries) == 0 {
		readMasterPassword()
	}

	var name string
	var password string

	fmt.Print("Enter name: ")
	fmt.Scanln(&name)

	filename := filepath.Join(dir, name)
	if _, err := os.Stat(filename); err == nil {
		fmt.Printf("Error: password for %s already exists\n", name)
		return nil
	}

	for {
		fmt.Print("Enter password: ")
		pwBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			fmt.Println()
			fmt.Println("Error reading password:", err)
			return nil
		}
		password = string(pwBytes)
		fmt.Println()

		fmt.Print("Confirm password: ")
		confirmBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			fmt.Println()
			fmt.Println("Error reading password:", err)
			return nil
		}
		fmt.Println()
		if password == string(confirmBytes) {
			break
		}
		fmt.Println("Password did not match, try again")
		fmt.Println()
	}

	credentials := map[string]string{"Name": name, "Password": password}
	return credentials
}
