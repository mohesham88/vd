package app

import (
	"encoding/json"
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
	if slices.Contains(ReadPasswordsLookup(), credentials.Name) {
		log.Printf("Error: password for %s already exists", credentials.Name)
		return false, nil
	}

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

func DeletePassword(credentialsName string) (bool, error) {
	if !slices.Contains(ReadPasswordsLookup(), credentialsName) {
		return false, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}

	dir := filepath.Join(home, ".local", "share", "vd", "passwords")
	filename := filepath.Join(dir, credentialsName)

	if _, err := os.Stat(filename); err != nil {
		UpdatePasswordsLookup(credentialsName, true)
		return false, nil
	}

	if err := os.Remove(filename); err != nil {
		return false, err
	}

	UpdatePasswordsLookup(credentialsName, true)

	return true, nil
}

func ChangePassword(credentialsName, newPassword string) (bool, error) {
	if !slices.Contains(ReadPasswordsLookup(), credentialsName) {
		return false, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}

	dir := filepath.Join(home, ".local", "share", "vd", "passwords")
	filename := filepath.Join(dir, credentialsName)

	data, err := createJSONObj(Credentials{Name: credentialsName, Password: newPassword})
	if err != nil {
		return false, err
	}

	encrypted, err := Encrypt(data)
	if err != nil {
		return false, err
	}

	if err := os.WriteFile(filename, encrypted, 0o600); err != nil {
		return false, err
	}

	return true, nil
}

func changePassword(targetPassword string) {
	newPassword, err := ReadPassword(true)
	if err != nil {
		return
	}

	success, err := ChangePassword(targetPassword, newPassword)
	if err != nil {
		log.Println("Error changing password:", err)
		return
	}

	if !success {
		log.Printf("Error: password for %s doesn't exist", targetPassword)
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
