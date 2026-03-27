package monitor

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ValentinDumas/website-status-checker/internal/checker"
	"github.com/ValentinDumas/website-status-checker/internal/config"
)

// newTestConfig creates a config pointing at the given test server URLs.
func newTestConfig(urls ...string) *config.Config {
	sites := make([]config.Site, len(urls))
	for i, u := range urls {
		sites[i] = config.Site{
			Name:          "Site" + string(rune('A'+i)),
			URL:           u,
			CheckInterval: 1, // 1 second for fast tests
		}
	}
	return &config.Config{
		Settings: config.Settings{
			CheckInterval:  1,
			RequestTimeout: 5,
		},
		Sites: sites,
	}
}

// ---------------------------------------------------------------------------
// Start / Stop lifecycle
// ---------------------------------------------------------------------------

func TestMonitor_StartStop(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := newTestConfig(server.URL)
	chk := checker.NewChecker(5 * time.Second)
	mon := NewMonitor(cfg, chk, nil, nil)

	mon.Start()
	// Give the monitor time to perform the initial check.
	time.Sleep(200 * time.Millisecond)
	mon.Stop()

	statuses := mon.GetStatuses()
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if !statuses[0].LatestResult.IsUp {
		t.Error("expected site to be up")
	}
}

// ---------------------------------------------------------------------------
// GetStatuses preserves config order
// ---------------------------------------------------------------------------

func TestMonitor_GetStatuses_Order(t *testing.T) {
	serverA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer serverA.Close()

	serverB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer serverB.Close()

	cfg := newTestConfig(serverA.URL, serverB.URL)
	chk := checker.NewChecker(5 * time.Second)
	mon := NewMonitor(cfg, chk, nil, nil)

	mon.Start()
	time.Sleep(200 * time.Millisecond)
	mon.Stop()

	statuses := mon.GetStatuses()
	if len(statuses) != 2 {
		t.Fatalf("expected 2 statuses, got %d", len(statuses))
	}
	if statuses[0].Site.Name != "SiteA" {
		t.Errorf("first status should be SiteA, got %s", statuses[0].Site.Name)
	}
	if statuses[1].Site.Name != "SiteB" {
		t.Errorf("second status should be SiteB, got %s", statuses[1].Site.Name)
	}
}

// ---------------------------------------------------------------------------
// RefreshAll
// ---------------------------------------------------------------------------

func TestMonitor_RefreshAll(t *testing.T) {
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := newTestConfig(server.URL)
	chk := checker.NewChecker(5 * time.Second)
	mon := NewMonitor(cfg, chk, nil, nil)

	// Don't Start() — just call RefreshAll() directly.
	mon.RefreshAll()

	if count.Load() != 1 {
		t.Errorf("expected 1 request from RefreshAll, got %d", count.Load())
	}
	statuses := mon.GetStatuses()
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if !statuses[0].LatestResult.IsUp {
		t.Error("expected site to be up after RefreshAll")
	}
}

// ---------------------------------------------------------------------------
// Status change detection
// ---------------------------------------------------------------------------

func TestMonitor_StatusChangeCallback(t *testing.T) {
	var isDown atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isDown.Load() {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	var callbackCalled atomic.Int32
	var lastStatus SiteStatus

	callback := func(status SiteStatus) {
		callbackCalled.Add(1)
		lastStatus = status
	}

	cfg := newTestConfig(server.URL)
	chk := checker.NewChecker(5 * time.Second)
	mon := NewMonitor(cfg, chk, callback, nil)

	// First check: site is up. No callback (first check, no previous state).
	mon.RefreshAll()
	if callbackCalled.Load() != 0 {
		t.Error("callback should not fire on first check")
	}

	// Second check: still up. No callback (no change).
	mon.RefreshAll()
	if callbackCalled.Load() != 0 {
		t.Error("callback should not fire when status is unchanged")
	}

	// Third check: site goes down. Callback should fire.
	isDown.Store(true)
	mon.RefreshAll()
	if callbackCalled.Load() != 1 {
		t.Errorf("callback count = %d, want 1 (site went down)", callbackCalled.Load())
	}
	if lastStatus.StatusChanged != true {
		t.Error("StatusChanged should be true")
	}
	if lastStatus.LatestResult.IsUp {
		t.Error("LatestResult.IsUp should be false (site is down)")
	}
	if !lastStatus.PreviousResult.IsUp {
		t.Error("PreviousResult.IsUp should be true (was up before)")
	}

	// Fourth check: site recovers. Callback should fire again.
	isDown.Store(false)
	mon.RefreshAll()
	if callbackCalled.Load() != 2 {
		t.Errorf("callback count = %d, want 2 (site recovered)", callbackCalled.Load())
	}
}

// ---------------------------------------------------------------------------
// ReloadConfig
// ---------------------------------------------------------------------------

func TestMonitor_ReloadConfig(t *testing.T) {
	serverA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer serverA.Close()

	serverB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer serverB.Close()

	cfg := newTestConfig(serverA.URL)
	chk := checker.NewChecker(5 * time.Second)
	mon := NewMonitor(cfg, chk, nil, nil)

	mon.Start()
	time.Sleep(200 * time.Millisecond)

	// Reload with a different config that has a new site.
	newCfg := newTestConfig(serverB.URL)
	newCfg.Sites[0].Name = "NewSite"
	mon.ReloadConfig(newCfg)
	time.Sleep(200 * time.Millisecond)
	mon.Stop()

	statuses := mon.GetStatuses()
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status after reload, got %d", len(statuses))
	}
	if statuses[0].Site.Name != "NewSite" {
		t.Errorf("expected site name %q, got %q", "NewSite", statuses[0].Site.Name)
	}
}

// ---------------------------------------------------------------------------
// Down site detection
// ---------------------------------------------------------------------------

func TestMonitor_DownSite(t *testing.T) {
	// Use a port that's not listening.
	cfg := &config.Config{
		Settings: config.Settings{
			CheckInterval:  1,
			RequestTimeout: 1,
		},
		Sites: []config.Site{
			{Name: "Down", URL: "http://127.0.0.1:1", CheckInterval: 1},
		},
	}
	chk := checker.NewChecker(1 * time.Second)
	mon := NewMonitor(cfg, chk, nil, nil)

	mon.RefreshAll()

	statuses := mon.GetStatuses()
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if statuses[0].LatestResult.IsUp {
		t.Error("expected site to be down")
	}
}
