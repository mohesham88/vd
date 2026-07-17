package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/term"
)

func readCredential() map[string]string {
	var name string
	var password string

	fmt.Print("Enter name: ")
	fmt.Scanln(&name)
	fmt.Print("Enter password: ")
	pwBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Println()
		fmt.Println("Error reading password:", err)
		return nil
	}

	password = string(pwBytes)
	fmt.Println()

	credentials := map[string]string{"Name": name, "Password": password}
	return credentials
}

func createJSONObj(data map[string]string) ([]byte, error) {
	jsonObj, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}
	return jsonObj, nil
}

func savePassword(credentials map[string]string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".local", "share", "vd", "passwords")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	filename := filepath.Join(dir, credentials["Name"]+".json")
	if _, err := os.Stat(filename); err == nil {
		fmt.Println("Error: password for", credentials["Name"], "already exists")
		return nil
	}

	data, err := createJSONObj(credentials)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0o600)
}

func getPassword(credentialsName string) {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory:", err)
		return
	}
	dir := filepath.Join(home, ".local", "share", "vd", "passwords")
	filename := filepath.Join(dir, credentialsName+".json")

	if _, err := os.Stat(filename); err != nil {
		fmt.Println("Error: password for", credentialsName, "doesn't exist")
		return
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println("Error")
		return
	}

	type Credentials struct {
		Name     string `json:"Name"`
		Password string `json:"Password"`
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		fmt.Println("Error parsing password file:", err)
		return
	}

	var clipCmd string
	switch runtime.GOOS {
	case "darwin":
		clipCmd = "pbcopy"
	case "linux":
		clipCmd = "xclip"
	default:
		fmt.Println("Error: This OS is not supported at the moment")
		return
	}
	cmd := exec.Command(clipCmd)
	cmd.Stdin = strings.NewReader(creds.Password)
	if err := cmd.Run(); err != nil {
		fmt.Println("Error copying to clipboard:", err)
		return
	}
	fmt.Printf("Password for `%s` copied to clipboard\n", creds.Name)
}

func deletePassword(credentialsName string) {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory:", err)
		return
	}
	dir := filepath.Join(home, ".local", "share", "vd", "passwords")
	filename := filepath.Join(dir, credentialsName+".json")

	if _, err := os.Stat(filename); err != nil {
		fmt.Println("Error: password for", credentialsName, "doesn't exist")
		return
	}

	fmt.Printf("Are you sure you want to delete the password for %s? (y/N): ", credentialsName)
	var confirm string
	fmt.Scanln(&confirm)
	if confirm != "y" && confirm != "Y" {
		return
	}

	if err := os.Remove(filename); err != nil {
		fmt.Println("Error deleting password:", err)
		return
	}
	fmt.Println("Password for", credentialsName, "deleted")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: vd <command>")
		fmt.Println("Commands:")
		fmt.Println("  add     Add a new password")
		fmt.Println("  get     Copy a password to clipboard")
		fmt.Println("  delete  Delete a stored password")
		return
	}

	switch os.Args[1] {
	case "add":
		obj := readCredential()
		if obj == nil {
			return
		}
		if err := savePassword(obj); err != nil {
			fmt.Println("Error saving password:", err)
		}
	case "get":
		if len(os.Args) < 3 {
			fmt.Println("Usage: vd get password_name")
			return
		}
		getPassword(os.Args[2])
	case "delete":
		if len(os.Args) < 3 {
			fmt.Println("Usage: vd delete password_name")
			return
		}
		deletePassword(os.Args[2])
	default:
		fmt.Println("Unknown command:", os.Args[1])
	}
}
