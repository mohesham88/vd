package app

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

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
