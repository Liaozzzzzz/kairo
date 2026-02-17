package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type CookieConfig struct {
	Enabled bool   `json:"enabled"`
	Source  string `json:"source"` // "browser" or "file"
	Browser string `json:"browser"`
	File    string `json:"file"`
}

type AppSettings struct {
	DownloadDir         string       `json:"downloadDir"`
	DownloadConcurrency int          `json:"downloadConcurrency"`
	MaxDownloadSpeed    *int         `json:"maxDownloadSpeed"` // MB/s
	Language            string       `json:"language"`
	ProxyUrl            string       `json:"proxyUrl"`
	UserAgent           string       `json:"userAgent"`
	Referer             string       `json:"referer"`
	GeoBypass           bool         `json:"geoBypass"`
	Cookie              CookieConfig `json:"cookie"`
	RSSCheckInterval    int          `json:"rssCheckInterval"` // Minutes
}

var (
	currentConfig AppSettings
	configMu      sync.RWMutex
)

func init() {
	currentConfig = AppSettings{
		DownloadConcurrency: 3,
		GeoBypass:           true,
		RSSCheckInterval:    60,
	}
}

func UpdateSettings(cfg AppSettings) {
	configMu.Lock()
	defer configMu.Unlock()
	currentConfig = cfg
	// Save settings to disk
	go SaveSettings()
}

func GetConfigPath() (string, error) {
	appDir, err := GetAppConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDir, "config.json"), nil
}

func SaveSettings() error {
	configMu.RLock()
	data, err := json.MarshalIndent(currentConfig, "", "  ")
	configMu.RUnlock()

	if err != nil {
		return err
	}

	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func LoadSettings() error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Set default download dir if config not exists
			configMu.Lock()
			if defaultDir, err := GetDefaultDownloadDir(); err == nil {
				currentConfig.DownloadDir = defaultDir
			}
			configMu.Unlock()
			return nil // Use defaults
		}
		return err
	}

	configMu.Lock()
	defer configMu.Unlock()
	if err := json.Unmarshal(data, &currentConfig); err != nil {
		return err
	}

	// Ensure default download dir
	if currentConfig.DownloadDir == "" {
		if defaultDir, err := GetDefaultDownloadDir(); err == nil {
			currentConfig.DownloadDir = defaultDir
		}
	}

	return nil
}

func GetSettings() AppSettings {
	configMu.RLock()
	defer configMu.RUnlock()
	return currentConfig
}

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

func GetMaxConcurrentDownloads() int {
	configMu.RLock()
	defer configMu.RUnlock()
	if currentConfig.DownloadConcurrency <= 0 {
		return 3
	}
	return currentConfig.DownloadConcurrency
}

func GetDownloadRateLimit() string {
	configMu.RLock()
	defer configMu.RUnlock()
	if currentConfig.MaxDownloadSpeed == nil || *currentConfig.MaxDownloadSpeed <= 0 {
		return ""
	}
	return fmt.Sprintf("%dM", *currentConfig.MaxDownloadSpeed)
}

func GetProxyUrl() string {
	configMu.RLock()
	defer configMu.RUnlock()
	return currentConfig.ProxyUrl
}

func GetUserAgent() string {
	configMu.RLock()
	defer configMu.RUnlock()
	return currentConfig.UserAgent
}

func GetReferer() string {
	configMu.RLock()
	defer configMu.RUnlock()
	return currentConfig.Referer
}

func GetGeoBypass() bool {
	configMu.RLock()
	defer configMu.RUnlock()
	return currentConfig.GeoBypass
}

func GetCookieArgs() []string {
	configMu.RLock()
	cookieConfig := currentConfig.Cookie
	configMu.RUnlock()

	if cookieConfig.Enabled {
		if cookieConfig.Source == "browser" && cookieConfig.Browser != "" {
			return []string{"--cookies-from-browser", cookieConfig.Browser}
		}
		if cookieConfig.Source == "file" && cookieConfig.File != "" {
			return []string{"--cookies", cookieConfig.File}
		}
	}

	return nil
}
