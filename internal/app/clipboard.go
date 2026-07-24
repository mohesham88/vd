package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"golang.design/x/clipboard"
)

var (
	clipboardOnce sync.Once
	clipboardErr  error
)

func CopyToClipboard(text string) error {
	if isWayland() {
		if err := wlCopy(text); err == nil {
			return nil
		} else if !isMissingWlCopy(err) {
			return err
		}
		return nil
	}

	clipboardOnce.Do(func() {
		clipboardErr = clipboard.Init()
	})
	if clipboardErr != nil {
		return fmt.Errorf("error initializing clipboard: %w", clipboardErr)
	}

	clipboard.Write(clipboard.FmtText, []byte(text))

	return nil
}

func isWayland() bool {
	return os.Getenv("WAYLAND_DISPLAY") != ""
}

func wlCopy(text string) error {
	cmd := exec.Command("wl-copy", "--type", "text/plain")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	_, writeErr := io.WriteString(stdin, text)
	closeErr := stdin.Close()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("wl-copy: %w", err)
	}
	if writeErr != nil {
		return fmt.Errorf("wl-copy: %w", writeErr)
	}

	return closeErr
}

func isMissingWlCopy(err error) bool {
	var execErr *exec.Error
	return errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound)
}
