// Package autostart manages automatic startup registration on Windows.
// It writes/removes a registry entry under HKCU\Software\Microsoft\Windows\CurrentVersion\Run
// so the application starts when the user logs in.
package autostart

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const (
	registryPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	appName      = "WebsiteStatusChecker"
)

// AutoStarter manages auto-start registration for the current platform.
type AutoStarter interface {
	// Enable registers the application to start on boot.
	Enable() error
	// Disable removes the auto-start registration.
	Disable() error
	// IsEnabled checks whether auto-start is currently registered.
	IsEnabled() bool
}

// WindowsAutoStarter implements AutoStarter using the Windows registry.
type WindowsAutoStarter struct{}

// NewWindowsAutoStarter creates a new WindowsAutoStarter.
func NewWindowsAutoStarter() *WindowsAutoStarter {
	return &WindowsAutoStarter{}
}

// Enable adds a registry entry to start the application on user login.
// It uses the path of the currently running executable.
func (w *WindowsAutoStarter) Enable() error {
	exePath, err := executablePath()
	if err != nil {
		return fmt.Errorf("getting executable path: %w", err)
	}

	key, _, err := registry.CreateKey(registry.CURRENT_USER, registryPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("opening registry key: %w", err)
	}
	defer key.Close()

	// Quote the path and add -windowsgui flag equivalent via ldflags at build time.
	value := fmt.Sprintf(`"%s"`, exePath)
	if err := key.SetStringValue(appName, value); err != nil {
		return fmt.Errorf("setting registry value: %w", err)
	}

	return nil
}

// Disable removes the auto-start registry entry.
func (w *WindowsAutoStarter) Disable() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, registryPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("opening registry key: %w", err)
	}
	defer key.Close()

	if err := key.DeleteValue(appName); err != nil {
		// If the value doesn't exist, that's fine — already disabled.
		return nil
	}

	return nil
}

// IsEnabled checks whether the auto-start registry entry exists.
func (w *WindowsAutoStarter) IsEnabled() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, registryPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()

	_, _, err = key.GetStringValue(appName)
	return err == nil
}

// executablePath returns the absolute path of the running executable.
func executablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Abs(exe)
}
