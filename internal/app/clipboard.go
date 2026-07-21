package app

import (
	"fmt"
	"sync"

	"golang.design/x/clipboard"
)

var (
	clipboardOnce sync.Once
	clipboardErr  error
)

func copyToClipboard(text string) error {
	clipboardOnce.Do(func() {
		clipboardErr = clipboard.Init()
	})
	if clipboardErr != nil {
		return fmt.Errorf("error initializing clipboard: %w", clipboardErr)
	}

	clipboard.Write(clipboard.FmtText, []byte(text))

	return nil
}
