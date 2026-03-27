// Package autostart manages automatic startup registration across platforms.
// It provides a unified AutoStarter interface with platform-specific
// implementations selected at compile time via Go build tags.
package autostart

import (
	"os"
	"path/filepath"
)

const appName = "WebsiteStatusChecker"

// AutoStarter manages auto-start registration for the current platform.
type AutoStarter interface {
	// Enable registers the application to start on boot/login.
	Enable() error
	// Disable removes the auto-start registration.
	Disable() error
	// IsEnabled checks whether auto-start is currently registered.
	IsEnabled() bool
}

// executablePath returns the absolute path of the running executable.
func executablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Abs(exe)
}
