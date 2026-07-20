package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
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
		fmt.Println("Error: password for", name, "already exists")
		return nil
	}

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

func savePassword(credentials map[string]string) (bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}
	dir := filepath.Join(home, ".local", "share", "vd", "passwords")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return false, err
	}

	filename := filepath.Join(dir, credentials["Name"])

	data, err := createJSONObj(credentials)
	if err != nil {
		return false, err
	}

	encrypted, err := encrypt(data)
	if err != nil {
		return false, err
	}

	os.WriteFile(filename, encrypted, 0o600)

	updatePasswordsLookup(credentials["Name"])

	return true, nil
}

func getPassword(credentialsName string) {
	if !slices.Contains(readPasswordsLookup(), credentialsName) {
		fmt.Println("Error: password for", credentialsName, "doesn't exist")
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
		fmt.Println("Error: password for", credentialsName, "doesn't exist")
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

	type Credentials struct {
		Name     string `json:"Name"`
		Password string `json:"Password"`
	}

	var creds Credentials
	if err := json.Unmarshal(decrypted, &creds); err != nil {
		fmt.Println("Error parsing password file:", err)
		return
	}

	if err := copyToClipboard(creds.Password); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Password for `%s` copied to clipboard\n", creds.Name)
}

func copyToClipboard(text string) error {
	var clipCmd string

	switch runtime.GOOS {
	case "darwin":
		clipCmd = "pbcopy"
	case "linux":
		clipCmd = "xclip"
	default:
		return fmt.Errorf("this OS is not supported at the moment")
	}

	cmd := exec.Command(clipCmd)
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error copying to clipboard: %w", err)
	}

	return nil
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

	updatePasswordsLookup(credentialsName)

	fmt.Println("Password for", credentialsName, "deleted")
}

func listCurrentPasswords() {
	for _, name := range readPasswordsLookup() {
		fmt.Println(name)
	}
}

func readPasswordsLookup() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory:", err)
		return nil
	}
	lookupPath := filepath.Join(home, ".local", "share", "vd", "passwords_lookup")

	data, err := os.ReadFile(lookupPath)
	if err != nil {
		fmt.Println("Error reading passwords_lookup:", err)
		return nil
	}

	decrypted, err := decrypt(data)
	if err != nil {
		fmt.Println("Error decrypting passwords_lookup:", err)
		return nil
	}

	var lookup map[string][]string
	if err := json.Unmarshal(decrypted, &lookup); err != nil {
		fmt.Println("Error parsing passwords_lookup:", err)
		return nil
	}

	return lookup["current_passwords"]
}

func GenerateGPGKey(name, email, passphrase string) error {
	if passphrase == "" {
		return fmt.Errorf("passphrase must not be empty")
	}
	batch := fmt.Sprintf(`Key-Type: RSA
Key-Length: 4096
Subkey-Type: RSA
Subkey-Length: 4096
Name-Real: %s
Name-Email: %s
Expire-Date: 1y
Passphrase: %s
%%commit
`, name, email, passphrase)

	cmd := exec.Command("gpg",
		"--batch",
		"--pinentry-mode", "loopback",
		"--generate-key",
	)
	cmd.Stdin = bytes.NewBufferString(batch)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gpg key generation failed: %v: %s", err, stderr.String())
	}
	return nil
}

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
		fmt.Println("Error: password for", emailPath, "already exists")
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

func updatePasswordsLookup(newPasswordName string, toBeDeleted ...bool) {
	toBeDeletedFlag := false
	if len(toBeDeleted) > 0 {
		toBeDeletedFlag = toBeDeleted[0]
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory:", err)
		return
	}

	vdDir := filepath.Join(home, ".local", "share", "vd")
	lookupPath := filepath.Join(vdDir, "passwords_lookup")

	plaintext, err := os.ReadFile(lookupPath)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Println("Error reading passwords_lookup:", err)
			return
		}

		if toBeDeletedFlag {
			return
		}

		lookup := map[string][]string{"current_passwords": {newPasswordName}}
		data, err := json.Marshal(lookup)
		if err != nil {
			fmt.Println("Error creating lookup JSON:", err)
			return
		}

		encrypted, err := encrypt(data)
		if err != nil {
			fmt.Println("Error encrypting lookup:", err)
			return
		}

		if err := os.MkdirAll(vdDir, 0o700); err != nil {
			fmt.Println("Error creating vd directory:", err)
			return
		}

		if err := os.WriteFile(lookupPath, encrypted, 0o600); err != nil {
			fmt.Println("Error writing passwords_lookup:", err)
			return
		}

		return
	}

	decrypted, err := decrypt(plaintext)
	if err != nil {
		fmt.Println("Error decrypting passwords_lookup:", err)
		return
	}

	var lookup map[string][]string
	if err := json.Unmarshal(decrypted, &lookup); err != nil {
		fmt.Println("Error parsing passwords_lookup:", err)
		return
	}

	if toBeDeletedFlag {
		if !slices.Contains(lookup["current_passwords"], newPasswordName) {
			fmt.Println("Password", newPasswordName, "not found in lookup")
		}

		lookup["current_passwords"] = slices.DeleteFunc(lookup["current_passwords"], func(name string) bool {
			return name == newPasswordName
		})

	} else {
		if slices.Contains(lookup["current_passwords"], newPasswordName) {
			fmt.Println("Password", newPasswordName, "already exists")
			return
		}

		lookup["current_passwords"] = append(lookup["current_passwords"], newPasswordName)
	}

	data, err := json.Marshal(lookup)
	if err != nil {
		fmt.Println("Error encoding lookup JSON:", err)
		return
	}

	encrypted, err := encrypt(data)
	if err != nil {
		fmt.Println("Error encrypting lookup:", err)
		return
	}

	if err := os.WriteFile(lookupPath, encrypted, 0o600); err != nil {
		fmt.Println("Error writing passwords_lookup:", err)
		return
	}
}

func generateRandomPassword() (string, error) {
	charset := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()"
	length := 20

	finalPassword := make([]byte, length)
	max := big.NewInt(int64(len(charset)))

	for i := range finalPassword {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		finalPassword[i] = charset[n.Int64()]
	}

	return string(finalPassword), nil
}

func encrypt(plaintext []byte) ([]byte, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	emailPath := filepath.Join(home, ".local", "share", "vd", "gpg_email.txt")
	recipientBytes, err := os.ReadFile(emailPath)
	if err != nil {
		return nil, fmt.Errorf("reading gpg_email.txt: %w", err)
	}

	recipient := strings.TrimSpace(string(recipientBytes))

	cmd := exec.Command("gpg",
		"--batch", "--yes",
		"--encrypt", "--recipient", recipient,
	)

	cmd.Stdin = bytes.NewReader(plaintext)

	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gpg encrypt: %v: %s", err, errBuf.String())
	}

	return out.Bytes(), nil
}

func decrypt(ciphertext []byte) ([]byte, error) {
	cmd := exec.Command("gpg", "--quiet", "--pinentry-mode", "loopback", "--decrypt")
	cmd.Stdin = bytes.NewReader(ciphertext)
	cmd.Stderr = os.Stderr

	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gpg decrypt failed (wrong passphrase or no key): %w", err)
	}

	return out.Bytes(), nil
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

		success, err := savePassword(obj)
		if err != nil {
			return
		}

		if success {
			fmt.Println("Password for", obj["Name"], "added successfully")
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
	case "register":
		register()
	case "ls":
		listCurrentPasswords()
	case "gen":
		newPassword, err := generateRandomPassword()
		if err != nil {
			fmt.Println("Error: ", err)
			return
		}

		err = copyToClipboard(newPassword)
		if err != nil {
			fmt.Println("Error: ", err)
			return
		}

		fmt.Println("New random password has been copied to clipboard")

	default:
		fmt.Println("Unknown command:", os.Args[1])
	}
}
