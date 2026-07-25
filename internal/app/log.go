package app

import (
	"log"
	"os"
	"path/filepath"
)

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	vdDir := filepath.Join(home, ".local", "share", "vd")
	if err := os.MkdirAll(vdDir, 0o700); err != nil {
		return
	}

	logPath := filepath.Join(vdDir, "logs.txt")

	if fi, err := os.Stat(logPath); err == nil && fi.Size() > 50<<20 {
		if err := os.Truncate(logPath, 0); err != nil {
			return
		}
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}

	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
