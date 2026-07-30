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
