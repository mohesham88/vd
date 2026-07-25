package app

import (
	"encoding/json"
	"fmt"
	"log"
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

func SavePassword(credentials Credentials) (bool, error) {
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

	encrypted, err := Encrypt(data)
	if err != nil {
		return false, err
	}

	os.WriteFile(filename, encrypted, 0o600)

	UpdatePasswordsLookup(credentials.Name)

	return true, nil
}

func GetPassword(credentialsName string) {
	if !slices.Contains(ReadPasswordsLookup(), credentialsName) {
		log.Printf("Error: password for %s doesn't exist", credentialsName)
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Println("Error getting home directory:", err)
		return
	}

	dir := filepath.Join(home, ".local", "share", "vd", "passwords")
	filename := filepath.Join(dir, credentialsName)

	if _, err := os.Stat(filename); err != nil {
		log.Printf("Error: password for %s doesn't exist", credentialsName)
		UpdatePasswordsLookup(credentialsName, true)
		return
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		log.Println("Error reading password file:", err)
		return
	}

	decrypted, err := Decrypt(data)
	if err != nil {
		log.Println("Error decrypting password file:", err)
		return
	}

	var creds Credentials
	if err := json.Unmarshal(decrypted, &creds); err != nil {
		log.Println("Error parsing password file:", err)
		return
	}

	if err := CopyToClipboard(creds.Password); err != nil {
		log.Println(err)
		return
	}
	log.Printf("Password for `%s` copied to clipboard", creds.Name)
}

func deletePassword(credentialsName string) {
	if !slices.Contains(ReadPasswordsLookup(), credentialsName) {
		log.Printf("Error: password for %s doesn't exist", credentialsName)
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Println("Error getting home directory:", err)
		return
	}

	dir := filepath.Join(home, ".local", "share", "vd", "passwords")
	filename := filepath.Join(dir, credentialsName)

	if _, err := os.Stat(filename); err != nil {
		log.Printf("Error: password for %s doesn't exist", credentialsName)
		UpdatePasswordsLookup(credentialsName, true)
		return
	}

	log.Printf("Are you sure you want to delete the password for %s? (y/N): ", credentialsName)
	var confirm string
	fmt.Scanln(&confirm)
	if confirm != "y" && confirm != "Y" {
		return
	}

	if err := os.Remove(filename); err != nil {
		log.Println("Error deleting password:", err)
		return
	}

	UpdatePasswordsLookup(credentialsName, true)

	log.Printf("Password for %s deleted", credentialsName)
}

func changePassword(targetPassword string) {
	if !slices.Contains(ReadPasswordsLookup(), targetPassword) {
		log.Printf("Error: password for %s doesn't exist", targetPassword)
		return
	}

	newPassword, err := ReadPassword(true)
	if err != nil {
		return
	}

	var creds Credentials
	creds.Name = targetPassword
	creds.Password = newPassword

	_, err = SavePassword(creds)
	if err != nil {
		return
	}

	log.Printf("Password for `%s` changed successfully", targetPassword)
}

func listCurrentPasswords() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Println("Error getting home directory:", err)
		return
	}

	passwordsDir := filepath.Join(home, ".local", "share", "vd", "passwords")

	for _, name := range ReadPasswordsLookup() {
		if _, err := os.Stat(filepath.Join(passwordsDir, name)); err != nil {
			if os.IsNotExist(err) {
				UpdatePasswordsLookup(name, true)
				continue
			}
			log.Println("Error checking password file:", err)
			continue
		}
		log.Println(name)
	}
}
