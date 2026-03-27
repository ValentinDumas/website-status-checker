// Package monitor runs background health checks for all configured sites
// and maintains an in-memory status store that other components (like the
// system tray) can read from.
package monitor

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/ValentinDumas/website-status-checker/internal/checker"
	"github.com/ValentinDumas/website-status-checker/internal/config"
)

// SiteStatus holds the current and previous check results for a single site,
// along with whether the status changed on the most recent check.
type SiteStatus struct {
	Site          config.Site
	LatestResult  checker.Result
	PreviousResult checker.Result
	StatusChanged bool // true when IsUp transitioned between checks
}

// StatusChangeCallback is called whenever a site's status transitions
// (up → down or down → up). It is NOT called for the initial check.
type StatusChangeCallback func(status SiteStatus)

// ConnectivityChangeCallback is called whenever the machine's internet
// connection state transitions (online <-> offline).
type ConnectivityChangeCallback func(isOnline bool)

// Monitor orchestrates periodic health checks for all configured sites.
// It is safe for concurrent access.
type Monitor struct {
	checker              *checker.Checker
	config               *config.Config
	onStatusChange       StatusChangeCallback
	onConnectivityChange ConnectivityChangeCallback

	mu       sync.RWMutex
	statuses map[string]*SiteStatus // keyed by site name
	isOnline bool                   // tracks if the machine has internet connectivity

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewMonitor creates a Monitor that uses the given checker and config.
// The optional onStatusChange callback is invoked whenever a site's up/down
// status transitions.
func NewMonitor(cfg *config.Config, chk *checker.Checker, onStatusChange StatusChangeCallback, onConnectivityChange ConnectivityChangeCallback) *Monitor {
	return &Monitor{
		checker:              chk,
		config:               cfg,
		onStatusChange:       onStatusChange,
		onConnectivityChange: onConnectivityChange,
		statuses:             make(map[string]*SiteStatus),
		isOnline:             true, // Assume true until proven otherwise
	}
}

// Start launches a background goroutine for each configured site.
// Each goroutine checks its site at the configured interval.
// Call Stop() to shut down all goroutines gracefully.
func (m *Monitor) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	// Synchronous first connectivity check.
	m.isOnline = m.checkInternetConnection()

	// Start continuous connectivity monitoring.
	m.wg.Add(1)
	go m.monitorConnectivity(ctx)

	for i := range m.config.Sites {
		site := m.config.Sites[i]
		interval := time.Duration(site.EffectiveCheckInterval(m.config.Settings.CheckInterval)) * time.Second

		m.wg.Add(1)
		go m.monitorSite(ctx, site, interval)
	}
}

// Stop cancels all monitoring goroutines and waits for them to exit.
func (m *Monitor) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()
}

// GetStatuses returns a snapshot of the current status for all monitored sites.
// The returned slice is a copy and safe to use without holding the lock.
func (m *Monitor) GetStatuses() []SiteStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]SiteStatus, 0, len(m.config.Sites))
	// Preserve config order instead of map iteration order.
	for _, site := range m.config.Sites {
		if status, ok := m.statuses[site.Name]; ok {
			result = append(result, *status)
		} else {
			// Return a placeholder so clients know the site is configured but pending.
			result = append(result, SiteStatus{Site: site})
		}
	}
	return result
}

// RefreshAll triggers an immediate check on all monitored sites.
// This runs synchronously — it blocks until all checks complete.
func (m *Monitor) RefreshAll() {
	var wg sync.WaitGroup
	for i := range m.config.Sites {
		site := m.config.Sites[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.checkAndUpdate(site)
		}()
	}
	wg.Wait()
}

// ReloadConfig stops all current goroutines, replaces the config, clears
// stale statuses, and restarts monitoring with the new configuration.
func (m *Monitor) ReloadConfig(cfg *config.Config) {
	m.Stop()

	m.mu.Lock()
	m.config = cfg
	// Clear statuses for sites that no longer exist in the new config.
	newSiteNames := make(map[string]bool, len(cfg.Sites))
	for _, site := range cfg.Sites {
		newSiteNames[site.Name] = true
	}
	for name := range m.statuses {
		if !newSiteNames[name] {
			delete(m.statuses, name)
		}
	}
	m.mu.Unlock()

	m.Start()
}

// monitorSite is the main loop for a single site. It runs in its own goroutine
// and checks the site once immediately, then at the configured interval.
func (m *Monitor) monitorSite(ctx context.Context, site config.Site, interval time.Duration) {
	defer m.wg.Done()

	// Check immediately on start.
	m.checkAndUpdate(site)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkAndUpdate(site)
		}
	}
}

// IsOnline returns true if the machine currently has internet connectivity.
func (m *Monitor) IsOnline() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isOnline
}

// checkInternetConnection attempts a lightweight TCP ping to a reliable host (Cloudflare DNS).
func (m *Monitor) checkInternetConnection() bool {
	conn, err := net.DialTimeout("tcp", "1.1.1.1:53", 3*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// monitorConnectivity periodically checks for network drops and triggers callbacks
// if the connection state changes.
func (m *Monitor) monitorConnectivity(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			currentOnline := m.checkInternetConnection()

			m.mu.Lock()
			previousOnline := m.isOnline
			m.isOnline = currentOnline
			m.mu.Unlock()

			if currentOnline != previousOnline && m.onConnectivityChange != nil {
				m.onConnectivityChange(currentOnline)
			}
		}
	}
}

// checkAndUpdate performs a health check and updates the status store.
// If the internet connection is currently offline, it skips the HTTP check
// to prevent false positives and does NOT execute the StatusChangeCallback.
func (m *Monitor) checkAndUpdate(site config.Site) {
	if !m.IsOnline() {
		return
	}

	result := m.checker.Check(site)

	m.mu.Lock()
	existing, hasExisting := m.statuses[site.Name]

	status := &SiteStatus{
		Site:         site,
		LatestResult: result,
	}

	if hasExisting {
		status.PreviousResult = existing.LatestResult
		status.StatusChanged = existing.LatestResult.IsUp != result.IsUp
	}

	m.statuses[site.Name] = status
	m.mu.Unlock()

	// Fire callback outside the lock to avoid deadlocks.
	if hasExisting && status.StatusChanged && m.onStatusChange != nil {
		m.onStatusChange(*status)
	}
}
