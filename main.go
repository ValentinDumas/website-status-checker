// Website Status Checker
//
// A lightweight system tray application that monitors your websites
// and shows their status as colored indicators in the taskbar.
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/getlantern/systray"

	"github.com/ValentinDumas/website-status-checker/internal/checker"
	"github.com/ValentinDumas/website-status-checker/internal/config"
	"github.com/ValentinDumas/website-status-checker/internal/monitor"
	"github.com/ValentinDumas/website-status-checker/internal/notify"
	"github.com/ValentinDumas/website-status-checker/internal/tray"
)
func main() {
	var actualConfigPath string

	if len(os.Args) > 1 {
		actualConfigPath = os.Args[1]
	}

	cfg, err := config.LoadConfig(actualConfigPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// For the tray manager, if we loaded the default config, we need its absolute path
	trayConfigPath := actualConfigPath
	if trayConfigPath == "" {
		trayConfigPath, err = config.GetConfigPath()
		if err != nil {
			log.Fatalf("Failed to resolve absolute config path: %v", err)
		}
	}

	fmt.Printf("Loaded %d sites from %s\n", len(cfg.Sites), trayConfigPath)

	timeout := time.Duration(cfg.Settings.RequestTimeout) * time.Second
	chk := checker.NewChecker(timeout)

	// Set up desktop notifications for status changes.
	var onStatusChange monitor.StatusChangeCallback
	if cfg.Settings.NotifyOnChange {
		notifier := notify.NewDesktopNotifier()
		onStatusChange = notify.StatusChangeHandler(notifier)
	}

	mon := monitor.NewMonitor(cfg, chk, onStatusChange)

	mgr := tray.NewManager(mon, trayConfigPath, config.LoadConfig)

	systray.Run(mgr.OnReady, mgr.OnExit)
}
