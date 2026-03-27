//go:build darwin

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{.Label}}</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.ExePath}}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
</dict>
</plist>
`

// New creates the platform-specific AutoStarter for macOS.
func New() AutoStarter {
	return &darwinAutoStarter{}
}

// darwinAutoStarter implements AutoStarter using macOS LaunchAgents.
// It creates/removes a plist file in ~/Library/LaunchAgents/.
type darwinAutoStarter struct{}

func (d *darwinAutoStarter) plistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", "com."+appName+".plist")
}

func (d *darwinAutoStarter) Enable() error {
	exePath, err := executablePath()
	if err != nil {
		return fmt.Errorf("getting executable path: %w", err)
	}

	tmpl, err := template.New("plist").Parse(plistTemplate)
	if err != nil {
		return fmt.Errorf("parsing plist template: %w", err)
	}

	f, err := os.Create(d.plistPath())
	if err != nil {
		return fmt.Errorf("creating plist file: %w", err)
	}
	defer f.Close()

	data := struct {
		Label   string
		ExePath string
	}{
		Label:   "com." + appName,
		ExePath: exePath,
	}

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("writing plist file: %w", err)
	}

	return nil
}

func (d *darwinAutoStarter) Disable() error {
	err := os.Remove(d.plistPath())
	if os.IsNotExist(err) {
		return nil // already disabled
	}
	return err
}

func (d *darwinAutoStarter) IsEnabled() bool {
	_, err := os.Stat(d.plistPath())
	return err == nil
}
