package app

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

var (
	copyOnce sync.Once
	copyCmd  []string
)

func CopyToClipboard(text string) error {
	writeOSC52(text)

	copyOnce.Do(func() {
		copyCmd = copyCommand()
	})

	if len(copyCmd) == 0 {
		return fmt.Errorf("no clipboard command found")
	}

	if copyCmd[0] == "osascript" {
		escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(text)
		return exec.Command("osascript", "-e", fmt.Sprintf(`set the clipboard to "%s"`, escaped)).Run()
	}

	cmd := exec.Command(copyCmd[0], copyCmd[1:]...)
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func ClipboardImage() (string, error) {
	f, err := os.CreateTemp("", "vd-qr-*.png")
	if err != nil {
		return "", err
	}
	f.Close()
	path := f.Name()

	failed := func() (string, error) {
		os.Remove(path)
		return "", fmt.Errorf("no image found in the clipboard")
	}

	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf("set f to open for access POSIX file \"%s\" with write permission\nset eof f to 0\nwrite (the clipboard as «class PNGf») to f\nclose access f", path)
		if err := exec.Command("osascript", "-e", script).Run(); err != nil {
			out, err := exec.Command("osascript", "-e", "POSIX path of (the clipboard as «class furl»)").Output()
			if err != nil {
				return failed()
			}

			copied := strings.TrimSpace(string(out))
			if _, err := os.Stat(copied); err != nil {
				return failed()
			}

			os.Remove(path)
			return copied, nil
		}
	case "linux":
		var out []byte
		for _, mime := range []string{"image/png", "image/jpeg"} {
			if os.Getenv("WAYLAND_DISPLAY") != "" {
				out, _ = exec.Command("wl-paste", "--type", mime).Output()
			} else {
				out, _ = exec.Command("xclip", "-selection", "clipboard", "-t", mime, "-o").Output()
			}
			if len(out) > 0 {
				break
			}
		}
		if len(out) == 0 {
			return failed()
		}
		if err := os.WriteFile(path, out, 0o600); err != nil {
			return "", err
		}
	default:
		return failed()
	}

	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
		return failed()
	}

	return path, nil
}

func writeOSC52(text string) {
	sequence := fmt.Sprintf("\x1b]52;c;%s\x07", base64.StdEncoding.EncodeToString([]byte(text)))
	if os.Getenv("TMUX") != "" || os.Getenv("STY") != "" {
		sequence = fmt.Sprintf("\x1bPtmux;\x1b%s\x1b\\", sequence)
	}
	os.Stdout.WriteString(sequence)
}

func copyCommand() []string {
	has := func(name string) bool {
		_, err := exec.LookPath(name)
		return err == nil
	}

	switch runtime.GOOS {
	case "darwin":
		if has("osascript") {
			return []string{"osascript"}
		}
	case "linux":
		if os.Getenv("WAYLAND_DISPLAY") != "" && has("wl-copy") {
			return []string{"wl-copy"}
		}
		if has("xclip") {
			return []string{"xclip", "-selection", "clipboard"}
		}
		if has("xsel") {
			return []string{"xsel", "--clipboard", "--input"}
		}
	}
	return nil
}
