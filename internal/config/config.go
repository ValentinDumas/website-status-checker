// Package config handles loading, validating, and reloading the YAML
// configuration file that defines which websites to monitor and how.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"

	"gopkg.in/yaml.v3"
)

// Default values for optional configuration fields.
const (
	DefaultCheckInterval  = 30 // seconds
	DefaultRequestTimeout = 10 // seconds
)

// Config is the top-level configuration structure loaded from sites.yaml.
type Config struct {
	Settings Settings `yaml:"settings"`
	Sites    []Site   `yaml:"sites"`
}

// Settings holds global defaults that apply to all sites unless overridden.
type Settings struct {
	CheckInterval  int  `yaml:"check_interval"`  // seconds between checks (default: 30)
	RequestTimeout int  `yaml:"request_timeout"` // seconds before a site is marked down (default: 10)
	NotifyOnChange bool `yaml:"notify_on_change"` // send desktop notification on status change
}

// Site represents a single website to monitor.
type Site struct {
	Name           string `yaml:"name"`            // human-readable display name
	URL            string `yaml:"url"`             // full URL to check (must include scheme)
	CheckInterval  int    `yaml:"check_interval"`  // per-site override (0 = use global default)
	ExpectedStatus int    `yaml:"expected_status"` // expected HTTP status code (0 = accept any 2xx)
}

// EffectiveCheckInterval returns the check interval for this site,
// falling back to the global default if no per-site override is set.
func (s *Site) EffectiveCheckInterval(globalInterval int) int {
	if s.CheckInterval > 0 {
		return s.CheckInterval
	}
	return globalInterval
}

// LoadConfig reads and parses the YAML configuration file at the given path.
// If path is empty, it uses the OS-specific native configuration directory.
// It applies default values for missing optional fields and validates all
// required fields before returning.
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		var err error
		path, err = GetConfigPath()
		if err != nil {
			return nil, err
		}
		if err := EnsureConfigExists(path); err != nil {
			return nil, err
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	applyDefaults(&cfg)

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &cfg, nil
}

// applyDefaults fills in zero-valued optional fields with sensible defaults.
func applyDefaults(cfg *Config) {
	if cfg.Settings.CheckInterval <= 0 {
		cfg.Settings.CheckInterval = DefaultCheckInterval
	}
	if cfg.Settings.RequestTimeout <= 0 {
		cfg.Settings.RequestTimeout = DefaultRequestTimeout
	}
}

// validate checks that all required configuration fields are present and valid.
// It collects all errors rather than failing on the first one, so the user
// can fix everything in a single pass.
func validate(cfg *Config) error {
	var errs []error

	if len(cfg.Sites) == 0 {
		errs = append(errs, errors.New("at least one site must be configured"))
	}

	for i, site := range cfg.Sites {
		prefix := fmt.Sprintf("sites[%d]", i)

		if site.Name == "" {
			errs = append(errs, fmt.Errorf("%s: name is required", prefix))
		}

		if site.URL == "" {
			errs = append(errs, fmt.Errorf("%s: url is required", prefix))
		} else if err := validateURL(site.URL); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", prefix, err))
		}

		if site.CheckInterval < 0 {
			errs = append(errs, fmt.Errorf("%s: check_interval must be positive", prefix))
		}

		if site.ExpectedStatus < 0 || site.ExpectedStatus > 599 {
			errs = append(errs, fmt.Errorf("%s: expected_status must be between 0 and 599", prefix))
		}
	}

	return errors.Join(errs...)
}

// validateURL checks that a URL string is well-formed and uses http or https.
func validateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url %q: %w", rawURL, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("url %q must use http or https scheme", rawURL)
	}
	if u.Host == "" {
		return fmt.Errorf("url %q must include a host", rawURL)
	}
	return nil
}
