package app

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func ExportPasswords() (string, error) {
	names := ReadPasswordsLookup()
	if len(names) == 0 {
		return "", fmt.Errorf("error: no passwords to export")
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	if err := w.Write([]string{"name", "password"}); err != nil {
		return "", err
	}

	for _, name := range names {
		creds, err := LoadCredentials(name)
		if err != nil {
			return "", err
		}

		if err := w.Write([]string{creds.Name, creds.Password}); err != nil {
			return "", err
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	filename := filepath.Join(home, fmt.Sprintf("passwords_%d.csv", time.Now().Unix()))
	if err := os.WriteFile(filename, buf.Bytes(), 0o600); err != nil {
		return "", err
	}

	return filename, nil
}
