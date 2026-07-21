package app

import (
	"fmt"

	"github.com/atotto/clipboard"
)

func copyToClipboard(text string) error {
	if err := clipboard.WriteAll(text); err != nil {
		return fmt.Errorf("error copying to clipboard: %w", err)
	}

	return nil
}
