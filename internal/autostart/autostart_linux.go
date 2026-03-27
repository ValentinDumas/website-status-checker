//go:build linux

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

const desktopEntryTemplate = `[Desktop Entry]
Type=Application
Name=%s
Exec=%s
X-GNOME-Autostart-enabled=true
Hidden=false
NoDisplay=false
`

// New creates the platform-specific AutoStarter for Linux.
func New() AutoStarter {
	return &linuxAutoStarter{}
}

// linuxAutoStarter implements AutoStarter using XDG Autostart.
// It creates/removes a .desktop file in ~/.config/autostart/.
type linuxAutoStarter struct{}

func (l *linuxAutoStarter) desktopFilePath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to ~/.config
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "autostart", appName+".desktop")
}

func (l *linuxAutoStarter) Enable() error {
	exePath, err := executablePath()
	if err != nil {
		return fmt.Errorf("getting executable path: %w", err)
	}

	dir := filepath.Dir(l.desktopFilePath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating autostart directory: %w", err)
	}

	content := fmt.Sprintf(desktopEntryTemplate, appName, exePath)
	if err := os.WriteFile(l.desktopFilePath(), []byte(content), 0644); err != nil {
		return fmt.Errorf("writing desktop file: %w", err)
	}

	return nil
}

func (l *linuxAutoStarter) Disable() error {
	err := os.Remove(l.desktopFilePath())
	if os.IsNotExist(err) {
		return nil // already disabled
	}
	return err
}

func (l *linuxAutoStarter) IsEnabled() bool {
	_, err := os.Stat(l.desktopFilePath())
	return err == nil
}
