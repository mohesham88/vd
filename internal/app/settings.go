package app

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Settings struct {
	ShowPassword bool `json:"showPassword"`
}

func settingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".local", "share", "vd", "settings.json"), nil
}

func LoadSettings() Settings {
	var settings Settings

	path, err := settingsPath()
	if err != nil {
		return settings
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return settings
	}

	json.Unmarshal(data, &settings)

	return settings
}

func SaveSettings(settings Settings) error {
	path, err := settingsPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}
