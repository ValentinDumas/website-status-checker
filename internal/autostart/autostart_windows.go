//go:build windows

package autostart

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

const registryPath = `Software\Microsoft\Windows\CurrentVersion\Run`

// New creates the platform-specific AutoStarter for Windows.
func New() AutoStarter {
	return &windowsAutoStarter{}
}

// windowsAutoStarter implements AutoStarter using the Windows registry.
// It writes/removes a value under HKCU\Software\Microsoft\Windows\CurrentVersion\Run.
type windowsAutoStarter struct{}

func (w *windowsAutoStarter) Enable() error {
	exePath, err := executablePath()
	if err != nil {
		return fmt.Errorf("getting executable path: %w", err)
	}

	key, _, err := registry.CreateKey(registry.CURRENT_USER, registryPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("opening registry key: %w", err)
	}
	defer key.Close()

	value := fmt.Sprintf(`"%s"`, exePath)
	if err := key.SetStringValue(appName, value); err != nil {
		return fmt.Errorf("setting registry value: %w", err)
	}

	return nil
}

func (w *windowsAutoStarter) Disable() error {
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

func (w *windowsAutoStarter) IsEnabled() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, registryPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()

	_, _, err = key.GetStringValue(appName)
	return err == nil
}
