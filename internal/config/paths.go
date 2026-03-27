package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

const appFolderName = "WebsiteStatusChecker"
const configFileName = "sites.yaml"

//go:embed default.yaml
var defaultYaml []byte

// GetConfigPath returns the absolute path to the OS-specific configuration file.
// It uses os.UserConfigDir() which resolves to:
// - Windows: %AppData%\WebsiteStatusChecker\sites.yaml
// - macOS: ~/Library/Application Support/WebsiteStatusChecker/sites.yaml
// - Linux: ~/.config/WebsiteStatusChecker/sites.yaml
func GetConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("getting user config dir: %w", err)
	}

	appDir := filepath.Join(configDir, appFolderName)
	return filepath.Join(appDir, configFileName), nil
}

// EnsureConfigExists checks if the configuration file is present at the given path.
// If it does not exist, it creates the app directories and writes the embedded
// default configuration exactly as it is compiled into the binary.
func EnsureConfigExists(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil // File already exists
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("stat config file: %w", err)
	}

	// File doesn't exist; create directory and file
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	if err := os.WriteFile(path, defaultYaml, 0644); err != nil {
		return fmt.Errorf("writing default config: %w", err)
	}

	return nil
}
