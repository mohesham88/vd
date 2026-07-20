package app

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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
