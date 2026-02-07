package config

import (
	"fmt"
	"os"
	"path/filepath"
)

func GetAppConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = home
	}
	appDir := filepath.Join(configDir, "Kairo")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return "", err
	}
	return appDir, nil
}

func GetStorePath() (string, error) {
	appDir, err := GetAppConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDir, "tasks.json"), nil
}

func GetLogPath(id string) string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		configDir = home
	}
	logDir := filepath.Join(configDir, "Kairo", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return ""
	}
	return filepath.Join(logDir, fmt.Sprintf("task_%s.log", id))
}

func GetDefaultDownloadDir() (string, error) {
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Downloads"), nil
}

func GetBinDir() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil || cfg == "" {
		home, _ := os.UserHomeDir()
		cfg = filepath.Join(home, ".config")
	}
	base := filepath.Join(cfg, "Kairo", "bin")
	_ = os.MkdirAll(base, 0o755)
	return base, nil
}
