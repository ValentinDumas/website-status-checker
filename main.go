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
	var configPath string
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	} else {
		var err error
		configPath, err = config.GetConfigPath()
		if err != nil {
			log.Fatalf("Failed to get config path: %v", err)
		}
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
	if cfg.Settings.NotifyOnChange {
		notifier := notify.NewDesktopNotifier()
		onStatusChange = notify.StatusChangeHandler(notifier)
	}

	mon := monitor.NewMonitor(cfg, chk, onStatusChange)

	mgr := tray.NewManager(mon, configPath, config.LoadConfig)

	systray.Run(mgr.OnReady, mgr.OnExit)
}
