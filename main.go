package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

	filename := filepath.Join(dir, credentials["Name"])
	if _, err := os.Stat(filename); err == nil {
		fmt.Println("Error: password for", credentials["Name"], "already exists")
		return nil
	}

	data, err := createJSONObj(credentials)
	if err != nil {
		return err
	}

	encrypted, err := encrypt(data)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, encrypted, 0o600)
}

func getPassword(credentialsName string) {
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
		if err := savePassword(obj); err != nil {
			fmt.Println("Error saving password:", err)
			return
		}
		fmt.Println("Password for", obj["Name"], "added successfully")
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
	default:
		fmt.Println("Unknown command:", os.Args[1])
	}
}
