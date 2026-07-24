package app

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const gpgPrefix = "\033[31m[GPG]\033[0m "

var (
	PassphraseCache  string
	NoTerminalPrompt bool
)

func printErr(args ...any) {
	if NoTerminalPrompt {
		return
	}
	fmt.Println(args...)
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

func newPrefixWriter(w io.Writer, prefix string) *prefixWriter {
	return &prefixWriter{w: w, prefix: []byte(prefix), atBOL: true}
}

func (p *prefixWriter) Write(b []byte) (int, error) {
	n := len(b)
	for len(b) > 0 {
		if p.atBOL {
			if _, err := p.w.Write(p.prefix); err != nil {
				return 0, err
			}
			p.atBOL = false
		}
		i := bytes.IndexByte(b, '\n')
		if i < 0 {
			if _, err := p.w.Write(b); err != nil {
				return 0, err
			}
			break
		}
		if _, err := p.w.Write(b[:i+1]); err != nil {
			return 0, err
		}
		p.atBOL = true
		b = b[i+1:]
	}
	return n, nil
}

func decrypt(ciphertext []byte) ([]byte, error) {
	var cached []byte
	if PassphraseCache != "" {
		cached = []byte(PassphraseCache)
	}

	var probeErr bytes.Buffer
	out, err := runGPGDecrypt(ciphertext, cached, &probeErr)
	if err == nil {
		return out.Bytes(), nil
	}

	if msg := probeErr.String(); strings.Contains(msg, "No secret key") ||
		strings.Contains(msg, "no valid OpenPGP data") {
		if !NoTerminalPrompt {
			os.Stderr.Write([]byte(prefixLines(msg)))
		}
		return nil, fmt.Errorf("gpg decrypt failed (wrong passphrase or no key)")
	}

	if NoTerminalPrompt {
		return nil, fmt.Errorf("gpg needs a passphrase but the terminal is busy")
	}

	passphrase, err := readSecret(gpgPrefix + "Enter Passphrase: ")
	if err != nil {
		return nil, err
	}

	out, err = runGPGDecrypt(ciphertext, []byte(passphrase), newPrefixWriter(os.Stderr, gpgPrefix))
	if err != nil {
		return nil, fmt.Errorf("gpg decrypt failed (wrong passphrase or no key): %w", err)
	}

	PassphraseCache = passphrase

	return out.Bytes(), nil
}

func runGPGDecrypt(ciphertext, passphrase []byte, stderr io.Writer) (*bytes.Buffer, error) {
	cmd := exec.Command("gpg", "--quiet", "--batch", "--decrypt")
	cmd.Stdin = bytes.NewReader(ciphertext)
	cmd.Stderr = stderr

	if passphrase != nil {
		r, w, err := os.Pipe()
		if err != nil {
			return nil, err
		}
		defer r.Close()

		// ExtraFiles[0] lands on fd 3 in the child.
		cmd.ExtraFiles = []*os.File{r}
		cmd.Args = append(cmd.Args, "--pinentry-mode", "loopback", "--passphrase-fd", "3")
		go func() {
			w.Write(append(passphrase, '\n'))
			w.Close()
		}()
	}

	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return &out, nil
}

func prefixLines(s string) string {
	var b bytes.Buffer
	newPrefixWriter(&b, gpgPrefix).Write([]byte(s))
	return b.String()
}
