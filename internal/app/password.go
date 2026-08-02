package app

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/ahmedhosssam/vd/internal/totp"
)

func createJSONObj(data Credentials) ([]byte, error) {
	jsonObj, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}
	return jsonObj, nil
}

func SavePassword(credentials Credentials) error {
	if slices.Contains(ReadPasswordsLookup(), credentials.Name) {
		log.Printf("Error: password for %s already exists", credentials.Name)
		return fmt.Errorf("error: password for %s already exists", credentials.Name)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Println("Error getting home directory:", err)
		return fmt.Errorf("error getting home directory: %w", err)
	}

	dir := filepath.Join(home, ".local", "share", "vd", "passwords")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		log.Println("Error creating passwords directory:", err)
		return fmt.Errorf("error creating passwords directory: %w", err)
	}

	filename := filepath.Join(dir, credentials.Name)

	data, err := createJSONObj(credentials)
	if err != nil {
		log.Println("Error encoding password JSON:", err)
		return fmt.Errorf("error encoding password JSON: %w", err)
	}

	encrypted, err := Encrypt(data)
	if err != nil {
		log.Println("Error encrypting password file:", err)
		return fmt.Errorf("error encrypting password file: %w", err)
	}

	if err := os.WriteFile(filename, encrypted, 0o600); err != nil {
		log.Println("Error writing password file:", err)
		return fmt.Errorf("error writing password file: %w", err)
	}

	UpdatePasswordsLookup(credentials.Name)

	if credentials.IsOTP {
		UpdateOTPLookup(credentials.Name)
	}

	log.Printf("Password for `%s` added successfully", credentials.Name)
	return nil
}

func LoadCredentials(credentialsName string) (Credentials, error) {
	var creds Credentials

	if !slices.Contains(ReadPasswordsLookup(), credentialsName) {
		log.Printf("Error: password for %s doesn't exist", credentialsName)
		return creds, fmt.Errorf("error: password for %s doesn't exist", credentialsName)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Println("Error getting home directory:", err)
		return creds, fmt.Errorf("error getting home directory: %w", err)
	}

	dir := filepath.Join(home, ".local", "share", "vd", "passwords")
	filename := filepath.Join(dir, credentialsName)

	if _, err := os.Stat(filename); err != nil {
		log.Printf("Error: password for %s doesn't exist", credentialsName)
		UpdatePasswordsLookup(credentialsName, true)
		return creds, fmt.Errorf("error: password for %s doesn't exist", credentialsName)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		log.Println("Error reading password file:", err)
		return creds, fmt.Errorf("error reading password file: %w", err)
	}

	decrypted, err := Decrypt(data)
	if err != nil {
		log.Println("Error decrypting password file:", err)
		return creds, fmt.Errorf("error decrypting password file: %w", err)
	}

	if err := json.Unmarshal(decrypted, &creds); err != nil {
		log.Println("Error parsing password file:", err)
		return creds, fmt.Errorf("error parsing password file: %w", err)
	}

	return creds, nil
}

func GetPassword(credentialsName string) (string, error) {
	creds, err := LoadCredentials(credentialsName)
	if err != nil {
		return "", err
	}

	if creds.IsOTP {
		code, err := totp.GetTotpCode(creds.Password, time.Now(), 6)
		if err != nil {
			log.Println("Error generating OTP code:", err)
			return "", fmt.Errorf("error generating OTP code: %w", err)
		}
		return code, nil
	}

	return creds.Password, nil
}

func CopyPasswordToClipboard(credentialsName string) error {
	value, err := GetPassword(credentialsName)
	if err != nil {
		return err
	}

	if err := CopyToClipboard(value); err != nil {
		log.Println(err)
		return err
	}

	log.Printf("Password for `%s` copied to clipboard", credentialsName)
	return nil
}

func DeletePassword(credentialsName string) error {
	if !slices.Contains(ReadPasswordsLookup(), credentialsName) {
		return fmt.Errorf("error: password for %s doesn't exist", credentialsName)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	dir := filepath.Join(home, ".local", "share", "vd", "passwords")
	filename := filepath.Join(dir, credentialsName)

	if _, err := os.Stat(filename); err != nil {
		UpdatePasswordsLookup(credentialsName, true)
		return fmt.Errorf("error: password for %s doesn't exist", credentialsName)
	}

	if err := os.Remove(filename); err != nil {
		return err
	}

	UpdatePasswordsLookup(credentialsName, true)
	UpdateOTPLookup(credentialsName, true)

	return nil
}

func ChangePassword(credentialsName, newPassword string) error {
	if !slices.Contains(ReadPasswordsLookup(), credentialsName) {
		return fmt.Errorf("error: password for %s doesn't exist", credentialsName)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	dir := filepath.Join(home, ".local", "share", "vd", "passwords")
	filename := filepath.Join(dir, credentialsName)

	data, err := createJSONObj(Credentials{Name: credentialsName, Password: newPassword, IsOTP: IsOTP(credentialsName)})
	if err != nil {
		return err
	}

	encrypted, err := Encrypt(data)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filename, encrypted, 0o600); err != nil {
		return err
	}

	return nil
}

func RenamePassword(oldName, newName string) error {
	if oldName == newName {
		return nil
	}

	if slices.Contains(ReadPasswordsLookup(), newName) {
		return fmt.Errorf("error: password for %s already exists", newName)
	}

	creds, err := LoadCredentials(oldName)
	if err != nil {
		return err
	}

	creds.Name = newName

	if err := SavePassword(creds); err != nil {
		return err
	}

	if err := DeletePassword(oldName); err != nil {
		return err
	}

	return nil
}

func changePassword(targetPassword string) error {
	if !slices.Contains(ReadPasswordsLookup(), targetPassword) {
		log.Printf("Error: password for %s doesn't exist", targetPassword)
		return fmt.Errorf("error: password for %s doesn't exist", targetPassword)
	}

	newPassword, err := ReadPassword(true)
	if err != nil {
		log.Println("Error reading password:", err)
		return fmt.Errorf("error reading password: %w", err)
	}

	if err := ChangePassword(targetPassword, newPassword); err != nil {
		log.Println("Error changing password:", err)
		return fmt.Errorf("error changing password: %w", err)
	}

	log.Printf("Password for `%s` changed successfully", targetPassword)
	return nil
}

func listCurrentPasswords() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory:", err)
		return
	}

	passwordsDir := filepath.Join(home, ".local", "share", "vd", "passwords")
	otpNames := ReadOTPLookup()

	for _, name := range ReadPasswordsLookup() {
		if _, err := os.Stat(filepath.Join(passwordsDir, name)); err != nil {
			if os.IsNotExist(err) {
				UpdatePasswordsLookup(name, true)
				continue
			}
			fmt.Println("Error checking password file:", err)
			continue
		}
		if slices.Contains(otpNames, name) {
			fmt.Println(name, "[OTP]")
			continue
		}
		fmt.Println(name)
	}
}
