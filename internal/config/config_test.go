package config

import (
	"os"
	"path/filepath"
	"testing"
)

// helper writes content to a temp YAML file and returns its path.
func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "sites.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	return path
}

// ---------------------------------------------------------------------------
// LoadConfig — happy paths
// ---------------------------------------------------------------------------

func TestLoadConfig_ValidFull(t *testing.T) {
	yaml := `
settings:
  check_interval: 60
  request_timeout: 5
  notify_on_change: true
sites:
  - name: "Site A"
    url: "https://example.com"
    expected_status: 200
    check_interval: 10
  - name: "Site B"
    url: "http://other.example.com"
`
	cfg, err := LoadConfig(writeTempConfig(t, yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Settings
	if cfg.Settings.CheckInterval != 60 {
		t.Errorf("CheckInterval = %d, want 60", cfg.Settings.CheckInterval)
	}
	if cfg.Settings.RequestTimeout != 5 {
		t.Errorf("RequestTimeout = %d, want 5", cfg.Settings.RequestTimeout)
	}
	if !cfg.Settings.NotifyOnChange {
		t.Error("NotifyOnChange = false, want true")
	}

	// Sites
	if len(cfg.Sites) != 2 {
		t.Fatalf("len(Sites) = %d, want 2", len(cfg.Sites))
	}

	site := cfg.Sites[0]
	if site.Name != "Site A" {
		t.Errorf("Sites[0].Name = %q, want %q", site.Name, "Site A")
	}
	if site.URL != "https://example.com" {
		t.Errorf("Sites[0].URL = %q, want %q", site.URL, "https://example.com")
	}
	if site.ExpectedStatus != 200 {
		t.Errorf("Sites[0].ExpectedStatus = %d, want 200", site.ExpectedStatus)
	}
	if site.CheckInterval != 10 {
		t.Errorf("Sites[0].CheckInterval = %d, want 10", site.CheckInterval)
	}
}

func TestLoadConfig_DefaultsApplied(t *testing.T) {
	yaml := `
sites:
  - name: "Minimal"
    url: "https://example.com"
`
	cfg, err := LoadConfig(writeTempConfig(t, yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Settings.CheckInterval != DefaultCheckInterval {
		t.Errorf("CheckInterval = %d, want default %d", cfg.Settings.CheckInterval, DefaultCheckInterval)
	}
	if cfg.Settings.RequestTimeout != DefaultRequestTimeout {
		t.Errorf("RequestTimeout = %d, want default %d", cfg.Settings.RequestTimeout, DefaultRequestTimeout)
	}
}

// ---------------------------------------------------------------------------
// LoadConfig — error paths
// ---------------------------------------------------------------------------

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/sites.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	content := `this is: [not: valid: yaml`
	_, err := LoadConfig(writeTempConfig(t, content))
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestLoadConfig_NoSites(t *testing.T) {
	yaml := `
settings:
  check_interval: 30
sites: []
`
	_, err := LoadConfig(writeTempConfig(t, yaml))
	if err == nil {
		t.Fatal("expected error for empty sites, got nil")
	}
}

func TestLoadConfig_MissingName(t *testing.T) {
	yaml := `
sites:
  - url: "https://example.com"
`
	_, err := LoadConfig(writeTempConfig(t, yaml))
	if err == nil {
		t.Fatal("expected error for missing name, got nil")
	}
}

func TestLoadConfig_MissingURL(t *testing.T) {
	yaml := `
sites:
  - name: "No URL"
`
	_, err := LoadConfig(writeTempConfig(t, yaml))
	if err == nil {
		t.Fatal("expected error for missing URL, got nil")
	}
}

func TestLoadConfig_InvalidURLScheme(t *testing.T) {
	yaml := `
sites:
  - name: "FTP Site"
    url: "ftp://example.com"
`
	_, err := LoadConfig(writeTempConfig(t, yaml))
	if err == nil {
		t.Fatal("expected error for ftp scheme, got nil")
	}
}

func TestLoadConfig_URLWithoutHost(t *testing.T) {
	yaml := `
sites:
  - name: "Bad URL"
    url: "https://"
`
	_, err := LoadConfig(writeTempConfig(t, yaml))
	if err == nil {
		t.Fatal("expected error for URL without host, got nil")
	}
}

func TestLoadConfig_NegativeCheckInterval(t *testing.T) {
	yaml := `
sites:
  - name: "Negative Interval"
    url: "https://example.com"
    check_interval: -5
`
	_, err := LoadConfig(writeTempConfig(t, yaml))
	if err == nil {
		t.Fatal("expected error for negative check_interval, got nil")
	}
}

func TestLoadConfig_InvalidExpectedStatus(t *testing.T) {
	yaml := `
sites:
  - name: "Bad Status"
    url: "https://example.com"
    expected_status: 999
`
	_, err := LoadConfig(writeTempConfig(t, yaml))
	if err == nil {
		t.Fatal("expected error for expected_status 999, got nil")
	}
}

func TestLoadConfig_MultipleErrors(t *testing.T) {
	yaml := `
sites:
  - name: ""
    url: ""
  - url: "ftp://bad"
`
	_, err := LoadConfig(writeTempConfig(t, yaml))
	if err == nil {
		t.Fatal("expected multiple validation errors, got nil")
	}
	// Should contain errors for both sites
	errStr := err.Error()
	if !containsSubstring(errStr, "sites[0]") {
		t.Errorf("error should mention sites[0], got: %s", errStr)
	}
	if !containsSubstring(errStr, "sites[1]") {
		t.Errorf("error should mention sites[1], got: %s", errStr)
	}
}

// ---------------------------------------------------------------------------
// EffectiveCheckInterval
// ---------------------------------------------------------------------------

func TestSite_EffectiveCheckInterval_Override(t *testing.T) {
	site := Site{CheckInterval: 10}
	if got := site.EffectiveCheckInterval(30); got != 10 {
		t.Errorf("EffectiveCheckInterval = %d, want 10 (per-site override)", got)
	}
}

func TestSite_EffectiveCheckInterval_Default(t *testing.T) {
	site := Site{CheckInterval: 0}
	if got := site.EffectiveCheckInterval(30); got != 30 {
		t.Errorf("EffectiveCheckInterval = %d, want 30 (global default)", got)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
