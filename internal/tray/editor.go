package tray

import (
	"fmt"
	"os/exec"
	"runtime"
)

// openEditor launches the user's default text editor to open the specified file.
// It uses OS-specific commands to achieve this cross-platform:
// - Windows: cmd /c start "" "path"
// - macOS: open "path"
// - Linux: xdg-open "path"
func openEditor(path string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// 'start' requires a window title string to prevent it from confusing a quoted path for a title.
		cmd = exec.Command("cmd", "/c", "start", "", path)
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open editor on %s: %w", runtime.GOOS, err)
	}

	return nil
}
