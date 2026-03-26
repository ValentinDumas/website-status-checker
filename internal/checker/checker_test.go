package checker

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ValentinDumas/website-status-checker/internal/config"
)

// ---------------------------------------------------------------------------
// Checker.Check — happy paths
// ---------------------------------------------------------------------------

func TestCheck_200OK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewChecker(5 * time.Second)
	result := c.Check(config.Site{Name: "Test", URL: server.URL})

	if !result.IsUp {
		t.Error("expected IsUp=true for 200 OK")
	}
	if result.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", result.StatusCode)
	}
	if result.ResponseTime <= 0 {
		t.Error("ResponseTime should be positive")
	}
	if result.Error != nil {
		t.Errorf("unexpected error: %v", result.Error)
	}
	if result.SiteName != "Test" {
		t.Errorf("SiteName = %q, want %q", result.SiteName, "Test")
	}
	if result.CheckedAt.IsZero() {
		t.Error("CheckedAt should not be zero")
	}
}

func TestCheck_201Created(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	c := NewChecker(5 * time.Second)
	result := c.Check(config.Site{Name: "Test", URL: server.URL})

	if !result.IsUp {
		t.Error("expected IsUp=true for 201 Created (2xx range)")
	}
}

func TestCheck_301Redirect(t *testing.T) {
	// Go's http.Client follows redirects by default, so the final response
	// should be from wherever the redirect lands.
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer final.Close()

	redirect := httptest.NewServer(http.RedirectHandler(final.URL, http.StatusMovedPermanently))
	defer redirect.Close()

	c := NewChecker(5 * time.Second)
	result := c.Check(config.Site{Name: "Redirect", URL: redirect.URL})

	if !result.IsUp {
		t.Error("expected IsUp=true after following redirect to 200")
	}
	if result.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200 (after redirect)", result.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Checker.Check — expected_status matching
// ---------------------------------------------------------------------------

func TestCheck_ExpectedStatus_Match(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewChecker(5 * time.Second)
	result := c.Check(config.Site{
		Name:           "Exact Match",
		URL:            server.URL,
		ExpectedStatus: 200,
	})

	if !result.IsUp {
		t.Error("expected IsUp=true when status matches expected_status")
	}
}

func TestCheck_ExpectedStatus_Mismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewChecker(5 * time.Second)
	result := c.Check(config.Site{
		Name:           "Mismatch",
		URL:            server.URL,
		ExpectedStatus: 204, // expect 204 but server returns 200
	})

	if result.IsUp {
		t.Error("expected IsUp=false when status doesn't match expected_status")
	}
}

func TestCheck_ExpectedStatus_401Trick(t *testing.T) {
	// The "expect 401" trick: server returns 401 (it's up, just not authenticated)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	c := NewChecker(5 * time.Second)
	result := c.Check(config.Site{
		Name:           "Auth Check",
		URL:            server.URL,
		ExpectedStatus: 401,
	})

	if !result.IsUp {
		t.Error("expected IsUp=true when 401 matches expected_status=401")
	}
}

// ---------------------------------------------------------------------------
// Checker.Check — error paths
// ---------------------------------------------------------------------------

func TestCheck_ServerError500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := NewChecker(5 * time.Second)
	result := c.Check(config.Site{Name: "500 Server", URL: server.URL})

	if result.IsUp {
		t.Error("expected IsUp=false for 500 Internal Server Error")
	}
	if result.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", result.StatusCode)
	}
	if result.Error != nil {
		t.Error("Error should be nil (server responded, just with 500)")
	}
}

func TestCheck_ConnectionRefused(t *testing.T) {
	c := NewChecker(2 * time.Second)
	// Port 1 is almost certainly not listening
	result := c.Check(config.Site{Name: "Refused", URL: "http://127.0.0.1:1"})

	if result.IsUp {
		t.Error("expected IsUp=false for connection refused")
	}
	if result.Error == nil {
		t.Error("expected non-nil Error for connection refused")
	}
	if result.StatusCode != 0 {
		t.Errorf("StatusCode = %d, want 0 (no response)", result.StatusCode)
	}
}

func TestCheck_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second) // sleep longer than the client timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewChecker(500 * time.Millisecond) // very short timeout
	result := c.Check(config.Site{Name: "Slow", URL: server.URL})

	if result.IsUp {
		t.Error("expected IsUp=false for timeout")
	}
	if result.Error == nil {
		t.Error("expected non-nil Error for timeout")
	}
}

func TestCheck_InvalidURL(t *testing.T) {
	c := NewChecker(2 * time.Second)
	result := c.Check(config.Site{Name: "Bad URL", URL: "://not-a-url"})

	if result.IsUp {
		t.Error("expected IsUp=false for invalid URL")
	}
	if result.Error == nil {
		t.Error("expected non-nil Error for invalid URL")
	}
}

// ---------------------------------------------------------------------------
// isHealthy — unit tests
// ---------------------------------------------------------------------------

func TestIsHealthy_2xxRange(t *testing.T) {
	tests := []struct {
		status int
		want   bool
	}{
		{200, true},
		{201, true},
		{204, true},
		{299, true},
		{199, false},
		{300, false},
		{404, false},
		{500, false},
	}

	for _, tt := range tests {
		got := isHealthy(tt.status, 0)
		if got != tt.want {
			t.Errorf("isHealthy(%d, 0) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestIsHealthy_ExactMatch(t *testing.T) {
	if !isHealthy(401, 401) {
		t.Error("isHealthy(401, 401) should be true")
	}
	if isHealthy(200, 204) {
		t.Error("isHealthy(200, 204) should be false")
	}
}
