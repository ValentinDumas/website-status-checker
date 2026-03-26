// Package checker performs HTTP health checks against configured websites.
// It sends GET requests and evaluates the response to determine whether
// a site is considered "up" or "down".
package checker

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ValentinDumas/website-status-checker/internal/config"
)

// Result holds the outcome of a single health check for one site.
type Result struct {
	SiteName     string        // human-readable name from config
	URL          string        // URL that was checked
	StatusCode   int           // HTTP status code (0 if connection failed)
	ResponseTime time.Duration // round-trip time for the request
	IsUp         bool          // true if the site is considered healthy
	Error        error         // non-nil if the request failed entirely
	CheckedAt    time.Time     // when the check was performed
}

// Checker performs HTTP health checks using a configured HTTP client.
type Checker struct {
	client *http.Client
}

// NewChecker creates a Checker with the given request timeout.
func NewChecker(timeout time.Duration) *Checker {
	return &Checker{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Check performs an HTTP GET request to the site's URL and returns the result.
// A site is considered "up" if:
//   - The request succeeds (no network error)
//   - The status code matches ExpectedStatus (if set), or is in the 2xx range
func (c *Checker) Check(site config.Site) Result {
	result := Result{
		SiteName:  site.Name,
		URL:       site.URL,
		CheckedAt: time.Now(),
	}

	start := time.Now()
	resp, err := c.client.Get(site.URL)
	result.ResponseTime = time.Since(start)

	if err != nil {
		result.Error = fmt.Errorf("request failed: %w", err)
		result.IsUp = false
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.IsUp = isHealthy(resp.StatusCode, site.ExpectedStatus)

	return result
}

// isHealthy determines whether an HTTP status code indicates a healthy site.
// If expectedStatus is set (> 0), it must match exactly.
// Otherwise, any 2xx status code is considered healthy.
func isHealthy(statusCode, expectedStatus int) bool {
	if expectedStatus > 0 {
		return statusCode == expectedStatus
	}
	return statusCode >= 200 && statusCode < 300
}
