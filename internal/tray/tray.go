// Package tray manages the system tray icon, tooltip, and menu.
// It reads statuses from the Monitor and updates the UI accordingly.
package tray

import (
	"fmt"
	"os"
	"time"

	"github.com/getlantern/systray"

	"github.com/ValentinDumas/website-status-checker/internal/autostart"
	"github.com/ValentinDumas/website-status-checker/internal/config"
	"github.com/ValentinDumas/website-status-checker/internal/monitor"
)

// StatusLevel represents the aggregate health of all monitored sites.
type StatusLevel int

const (
	StatusAllUp      StatusLevel = iota // all sites are up
	StatusPartialDown                   // some sites are down
	StatusAllDown                       // all sites are down
	StatusUnknown                       // no checks performed yet
)

// 8-Spacing UI Rule (En Space U+2002) translates to ~8px visually.
const space8px = "\u2002"

// iconTheme defines the dictionary of UI string icons.
type iconTheme struct {
	Up              string
	Down            string
	Unknown         string
	NetworkOnline   string
	NetworkOffline  string
	UtilityEdit     string
	UtilityRefresh  string
	UtilityReload   string
	UtilityAutoOn   string
	UtilityAutoOff  string
	UtilityQuit     string
}

// currentTheme drives all textural graphics using Theme 1 (Vibrant Emojis).
var currentTheme = iconTheme{
	Up:              "🟢",
	Down:            "🔴",
	Unknown:         "⚪",
	NetworkOnline:   "📶",
	NetworkOffline:  "🚫",
	UtilityEdit:     "✏️",
	UtilityRefresh:  "🔄",
	UtilityReload:   "📄",
	UtilityAutoOn:   "🟢",
	UtilityAutoOff:  "⚪",
	UtilityQuit:     "❌",
}

// Icons embedded as Go byte slices.
// These are minimal 16x16 ICO files generated programmatically.
var (
	iconGreen  = generateICO(0, 200, 83)    // #00C853 — green
	iconYellow = generateICO(255, 193, 7)   // #FFC107 — amber
	iconRed    = generateICO(244, 67, 54)   // #F44336 — red
	iconGray   = generateICO(158, 158, 158) // #9E9E9E — unknown/loading
)

// Manager coordinates the system tray UI with the monitoring engine.
type Manager struct {
	monitor    *monitor.Monitor
	configPath string
	loadConfig func(string) (*config.Config, error)

	// Menu items — stored for dynamic updates.
	internetItem   *systray.MenuItem
	siteItems      []*systray.MenuItem
	editConfigItem *systray.MenuItem
	refreshItem    *systray.MenuItem
	reloadItem    *systray.MenuItem
	autostartItem *systray.MenuItem
	quitItem      *systray.MenuItem
	autoStarter   autostart.AutoStarter
}

// NewManager creates a tray Manager wired to the given monitor.
// loadConfig is the function used to reload configuration on demand.
func NewManager(mon *monitor.Monitor, configPath string, loadConfig func(string) (*config.Config, error)) *Manager {
	return &Manager{
		monitor:     mon,
		configPath:  configPath,
		loadConfig:  loadConfig,
		autoStarter: autostart.New(),
	}
}

// OnReady is called by systray.Run when the tray is ready.
// It sets up the initial icon, menu, and starts the UI update loop.
func (m *Manager) OnReady() {
	systray.SetIcon(iconGray)
	systray.SetTooltip("Website Status: checking...")

	m.buildMenu()
	m.monitor.Start()

	go m.updateLoop()
	go m.handleMenuClicks()
	go m.watchConfigLoop()
}

// OnExit is called by systray.Run when the tray is shutting down.
func (m *Manager) OnExit() {
	m.monitor.Stop()
}

// buildMenu creates the tray menu structure.
// Site items are created based on the current monitor config.
func (m *Manager) buildMenu() {
	m.internetItem = systray.AddMenuItem(fmt.Sprintf("%s%sNetwork: Connected", currentTheme.NetworkOnline, space8px), "Machine internet connectivity")
	m.internetItem.Disable()
	systray.AddSeparator()

	statuses := m.monitor.GetStatuses()

	// If we have existing site items from a previous build, we can't remove
	// them (systray limitation), so we hide old ones and create new ones.
	for _, item := range m.siteItems {
		item.Hide()
	}

	m.siteItems = make([]*systray.MenuItem, len(statuses))
	for i, s := range statuses {
		m.siteItems[i] = systray.AddMenuItem(formatSiteLabel(s), s.Site.URL)
		m.siteItems[i].Disable() // informational, not clickable
	}

	// If no statuses yet (first run), create items from monitor's config.
	if len(statuses) == 0 {
		// We'll populate them on the first update loop tick.
	}

	systray.AddSeparator()
	m.editConfigItem = systray.AddMenuItem(fmt.Sprintf("%s%sEdit Configuration", currentTheme.UtilityEdit, space8px), "Open config file in text editor")
	m.refreshItem = systray.AddMenuItem(fmt.Sprintf("%s%sRefresh Now", currentTheme.UtilityRefresh, space8px), "Check all sites immediately")
	m.reloadItem = systray.AddMenuItem(fmt.Sprintf("%s%sReload Config", currentTheme.UtilityReload, space8px), "Reload sites.yaml without restarting")
	systray.AddSeparator()
	m.autostartItem = systray.AddMenuItem(m.autostartLabel(), "Start application on Windows login")
	systray.AddSeparator()
	m.quitItem = systray.AddMenuItem(fmt.Sprintf("%s%sQuit", currentTheme.UtilityQuit, space8px), "Exit Website Status Checker")
}

// handleMenuClicks listens for menu item clicks in a blocking loop.
func (m *Manager) handleMenuClicks() {
	for {
		select {
		case <-m.editConfigItem.ClickedCh:
			m.handleEditConfig()
		case <-m.refreshItem.ClickedCh:
			go m.monitor.RefreshAll()
		case <-m.reloadItem.ClickedCh:
			m.handleReloadConfig()
		case <-m.autostartItem.ClickedCh:
			m.handleAutostartToggle()
		case <-m.quitItem.ClickedCh:
			systray.Quit()
			return
		}
	}
}

// handleEditConfig opens the configuration file in the default OS text editor.
func (m *Manager) handleEditConfig() {
	if err := openEditor(m.configPath); err != nil {
		systray.SetTooltip(fmt.Sprintf("Failed to open editor: %v", err))
	}
}

// handleAutostartToggle enables or disables auto-start on boot.
func (m *Manager) handleAutostartToggle() {
	if m.autoStarter.IsEnabled() {
		if err := m.autoStarter.Disable(); err != nil {
			systray.SetTooltip(fmt.Sprintf("Auto-start error: %v", err))
			return
		}
	} else {
		if err := m.autoStarter.Enable(); err != nil {
			systray.SetTooltip(fmt.Sprintf("Auto-start error: %v", err))
			return
		}
	}
	m.autostartItem.SetTitle(m.autostartLabel())
}

// autostartLabel returns the display label for the auto-start menu item.
func (m *Manager) autostartLabel() string {
	if m.autoStarter.IsEnabled() {
		return fmt.Sprintf("%s%sStart on Boot (enabled)", currentTheme.UtilityAutoOn, space8px)
	}
	return fmt.Sprintf("%s%sStart on Boot (disabled)", currentTheme.UtilityAutoOff, space8px)
}

// updateLoop periodically reads statuses from the monitor and updates
// the tray icon, tooltip, and menu item labels.
func (m *Manager) updateLoop() {
	// Small delay to let the initial checks complete.
	time.Sleep(500 * time.Millisecond)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		m.updateUI()
		<-ticker.C
	}
}

// updateUI reads current statuses and refreshes the tray icon, tooltip,
// and per-site menu item labels.
func (m *Manager) updateUI() {
	statuses := m.monitor.GetStatuses()
	if len(statuses) == 0 {
		return
	}

	isOnline := m.monitor.IsOnline()

	if !isOnline {
		m.internetItem.SetTitle(fmt.Sprintf("%s%sNetwork: Offline", currentTheme.NetworkOffline, space8px))
		systray.SetIcon(iconGray)
		systray.SetTooltip("Website Status: Machine Offline")
	} else {
		m.internetItem.SetTitle(fmt.Sprintf("%s%sNetwork: Connected", currentTheme.NetworkOnline, space8px))
		level := aggregateStatus(statuses)

		// Update icon.
		switch level {
		case StatusAllUp:
			systray.SetIcon(iconGreen)
		case StatusPartialDown:
			systray.SetIcon(iconYellow)
		case StatusAllDown:
			systray.SetIcon(iconRed)
		default:
			systray.SetIcon(iconGray)
		}

		// Update tooltip.
		upCount := 0
		for _, s := range statuses {
			if s.LatestResult.IsUp {
				upCount++
			}
		}
		systray.SetTooltip(fmt.Sprintf("Website Status: %d/%d sites up", upCount, len(statuses)))
	}

	// Update site menu items.
	for i, s := range statuses {
		if i < len(m.siteItems) {
			m.siteItems[i].SetTitle(formatSiteLabel(s))
			m.siteItems[i].SetTooltip(s.Site.URL)
		}
	}
}

// handleReloadConfig reloads the YAML config and reconfigures the monitor.
func (m *Manager) handleReloadConfig() {
	cfg, err := m.loadConfig(m.configPath)
	if err != nil {
		// Show error in tooltip — we can't pop up a dialog from systray easily.
		systray.SetTooltip(fmt.Sprintf("Config reload error: %v", err))
		return
	}
	m.monitor.ReloadConfig(cfg)

	// Rebuild menu items for the new config.
	// Wait a moment for initial checks to complete, then update.
	go func() {
		time.Sleep(1 * time.Second)
		m.rebuildSiteItems()
	}()
}

// watchConfigLoop periodically checks if the configuration file has been modified
// on disk and automatically reloads it if changes are detected.
func (m *Manager) watchConfigLoop() {
	var lastModTime time.Time

	// Get initial mod time if file exists
	if info, err := os.Stat(m.configPath); err == nil {
		lastModTime = info.ModTime()
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		info, err := os.Stat(m.configPath)
		if err != nil {
			continue // file might be temporarily locked or missing during a save
		}

		if info.ModTime().After(lastModTime) {
			lastModTime = info.ModTime()
			m.handleReloadConfig()
		}
	}
}

// rebuildSiteItems updates the site menu items after a config reload.
func (m *Manager) rebuildSiteItems() {
	statuses := m.monitor.GetStatuses()

	// Hide excess old items.
	for i := len(statuses); i < len(m.siteItems); i++ {
		m.siteItems[i].Hide()
	}

	// Update or create items.
	for i, s := range statuses {
		if i < len(m.siteItems) {
			m.siteItems[i].SetTitle(formatSiteLabel(s))
			m.siteItems[i].SetTooltip(s.Site.URL)
			m.siteItems[i].Show()
		} else {
			item := systray.AddMenuItem(formatSiteLabel(s), s.Site.URL)
			item.Disable()
			m.siteItems = append(m.siteItems, item)
		}
	}
}

// formatSiteLabel builds the display string for a site menu item.
// Implements 8-spacing rule (En Space) between icon, text, and metadata.
func formatSiteLabel(s monitor.SiteStatus) string {
	indicator := currentTheme.Up
	if !s.LatestResult.IsUp {
		indicator = currentTheme.Down
	}

	if s.LatestResult.CheckedAt.IsZero() {
		return fmt.Sprintf("%s%s%s%s(checking...)", currentTheme.Unknown, space8px, s.Site.Name, space8px)
	}

	if s.LatestResult.Error != nil {
		return fmt.Sprintf("%s%s%s%s(error)", indicator, space8px, s.Site.Name, space8px)
	}

	ms := s.LatestResult.ResponseTime.Milliseconds()
	return fmt.Sprintf("%s%s%s%s(%dms)", indicator, space8px, s.Site.Name, space8px, ms)
}

// aggregateStatus determines the overall status level across all sites.
func aggregateStatus(statuses []monitor.SiteStatus) StatusLevel {
	if len(statuses) == 0 {
		return StatusUnknown
	}

	upCount := 0
	for _, s := range statuses {
		if s.LatestResult.IsUp {
			upCount++
		}
	}

	switch {
	case upCount == len(statuses):
		return StatusAllUp
	case upCount == 0:
		return StatusAllDown
	default:
		return StatusPartialDown
	}
}
