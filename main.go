// Website Status Checker
//
// A lightweight system tray application that monitors your websites
// and shows their status as colored indicators in the taskbar.
package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/getlantern/systray"

	"github.com/ValentinDumas/website-status-checker/internal/checker"
	"github.com/ValentinDumas/website-status-checker/internal/config"
	"github.com/ValentinDumas/website-status-checker/internal/monitor"
	"github.com/ValentinDumas/website-status-checker/internal/notify"
	"github.com/ValentinDumas/website-status-checker/internal/tray"
)
const appConfigDirName = "WebsiteStatusChecker"
const configFileName = "sites.yaml"

//go:embed sites.yaml
var defaultSitesYAML []byte

func getConfigPath() (string, error) {
	if len(os.Args) > 1 {
		return os.Args[1], nil
	}

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to current directory if UserConfigDir is not available
		return configFileName, nil
	}

	appConfigDir := filepath.Join(userConfigDir, appConfigDirName)
	if err := os.MkdirAll(appConfigDir, 0755); err != nil {
		return "", fmt.Errorf("creating config directory: %w", err)
	}

	configPath := filepath.Join(appConfigDir, configFileName)

	// Create default config file if it doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.WriteFile(configPath, defaultSitesYAML, 0644); err != nil {
			return "", fmt.Errorf("writing default config file: %w", err)
		}
	}

	return configPath, nil
}

func main() {
	configPath, err := getConfigPath()
	if err != nil {
		log.Fatalf("Failed to initialize config path: %v", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Printf("Loaded %d sites from %s\n", len(cfg.Sites), configPath)

	timeout := time.Duration(cfg.Settings.RequestTimeout) * time.Second
	chk := checker.NewChecker(timeout)

	// Set up desktop notifications for status changes.
	var onStatusChange monitor.StatusChangeCallback
	var onConnectivityChange monitor.ConnectivityChangeCallback
	
	if cfg.Settings.NotifyOnChange {
		notifier := notify.NewDesktopNotifier()
		onStatusChange = notify.StatusChangeHandler(notifier)
		
		onConnectivityChange = func(isOnline bool) {
			if isOnline {
				_ = notifier.Send("🟢 Online", "Machine connected to the internet")
			} else {
				_ = notifier.Send("🔌 Offline", "Machine lost internet connection. Checks paused.")
			}
		}
	}

	mon := monitor.NewMonitor(cfg, chk, onStatusChange, onConnectivityChange)

	// For the tray manager, we pass the path so it can be opened/reloaded by the user.
	mgr := tray.NewManager(mon, configPath, config.LoadConfig)

	systray.Run(mgr.OnReady, mgr.OnExit)
}
