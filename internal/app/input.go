package app

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"bufio"
	"golang.org/x/term"
)

func ReadSecret(prompt string) (string, error) {
	fmt.Print(prompt)
	pwBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		log.Println("Error reading passphrase:", err)
		return "", err
	}
	return string(pwBytes), nil
}

func Confirm(prompt string) bool {
	fmt.Printf("%s (Y/n): ", prompt)

	var answer string
	fmt.Scanln(&answer)

	answer = strings.TrimSpace(answer)

	return answer == "Y"
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
		fmt.Println()
		if err != nil {
			fmt.Println("Error reading password:", err)
			return "", err
		}
		password = string(pwBytes)

		fmt.Print("Confirm password: ")
		confirmBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			fmt.Println("Error reading password:", err)
			return "", err
		}
		if password == string(confirmBytes) {
			break
		}
		fmt.Println("Password did not match, try again")
	}

	return password, nil
}

func ReadCredential() *Credentials {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory:", err)
		return nil
	}
	passwordsDir := filepath.Join(home, ".local", "share", "vd", "passwords")

	emailPath := filepath.Join(home, ".local", "share", "vd", "gpg_email.txt")
	emailBytes, err := os.ReadFile(emailPath)
	if err != nil || len(strings.TrimSpace(string(emailBytes))) == 0 {
		fmt.Println("No GPG key registered. Run `vd register` first.")
		return nil
	}

	fmt.Print("Enter name: ")
	reader := bufio.NewReader(os.Stdin)
	name, err := reader.ReadString('\n')
	if err != nil {
			fmt.Println("Error reading name:", err)
			return nil
	}
	name = strings.TrimSpace(name)

	filename := filepath.Join(passwordsDir, name)
	if _, err := os.Stat(filename); err == nil {
		fmt.Printf("Error: password for %s already exists\n", name)
		return nil
	}

	password, err := ReadPassword()
	if err != nil {
		return nil
	}

	return &Credentials{Name: name, Password: password}
}
