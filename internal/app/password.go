package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

func createJSONObj(data Credentials) ([]byte, error) {
	jsonObj, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}
	return jsonObj, nil
}

func savePassword(credentials Credentials) (bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}
	dir := filepath.Join(home, ".local", "share", "vd", "passwords")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return false, err
	}

	filename := filepath.Join(dir, credentials.Name)

	data, err := createJSONObj(credentials)
	if err != nil {
		return false, err
	}

	encrypted, err := encrypt(data)
	if err != nil {
		return false, err
	}

	os.WriteFile(filename, encrypted, 0o600)

	updatePasswordsLookup(credentials.Name)

	return true, nil
}

func GetPassword(credentialsName string) {
	if !slices.Contains(readPasswordsLookup(), credentialsName) {
		fmt.Printf("Error: password for %s doesn't exist\n", credentialsName)
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory:", err)
		return
	}

	dir := filepath.Join(home, ".local", "share", "vd", "passwords")
	filename := filepath.Join(dir, credentialsName)

	if _, err := os.Stat(filename); err != nil {
		fmt.Printf("Error: password for %s doesn't exist\n", credentialsName)
		updatePasswordsLookup(credentialsName, true)
		return
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println("Error reading password file:", err)
		return
	}

	decrypted, err := decrypt(data)
	if err != nil {
		fmt.Println("Error decrypting password file:", err)
		return
	}

	var creds Credentials
	if err := json.Unmarshal(decrypted, &creds); err != nil {
		fmt.Println("Error parsing password file:", err)
		return
	}

	if err := CopyToClipboard(creds.Password); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Password for `%s` copied to clipboard\n", creds.Name)
}

func deletePassword(credentialsName string) {
	if !slices.Contains(readPasswordsLookup(), credentialsName) {
		fmt.Printf("Error: password for %s doesn't exist\n", credentialsName)
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory:", err)
		return
	}

	dir := filepath.Join(home, ".local", "share", "vd", "passwords")
	filename := filepath.Join(dir, credentialsName)

	if _, err := os.Stat(filename); err != nil {
		fmt.Printf("Error: password for %s doesn't exist\n", credentialsName)
		updatePasswordsLookup(credentialsName, true)
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

	updatePasswordsLookup(credentialsName, true)

	fmt.Printf("Password for %s deleted\n", credentialsName)
}

func changePassword(targetPassword string) {
	if !slices.Contains(readPasswordsLookup(), targetPassword) {
		fmt.Printf("Error: password for %s doesn't exist\n", targetPassword)
		return
	}

	newPassword, err := readPassword(true)
	if err != nil {
		return
	}

	var creds Credentials
	creds.Name = targetPassword
	creds.Password = newPassword

	_, err = savePassword(creds)
	if err != nil {
		return
	}

	fmt.Printf("Password for `%s` changed successfully\n", targetPassword)
}

func listCurrentPasswords() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory:", err)
		return
	}

	passwordsDir := filepath.Join(home, ".local", "share", "vd", "passwords")

	for _, name := range readPasswordsLookup() {
		if _, err := os.Stat(filepath.Join(passwordsDir, name)); err != nil {
			if os.IsNotExist(err) {
				updatePasswordsLookup(name, true)
				continue
			}
			fmt.Println("Error checking password file:", err)
			continue
		}
		fmt.Println(name)
	}
}
