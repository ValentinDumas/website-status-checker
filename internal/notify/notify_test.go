package notify

import (
	"fmt"
	"testing"
	"time"

	"github.com/ValentinDumas/website-status-checker/internal/checker"
	"github.com/ValentinDumas/website-status-checker/internal/config"
	"github.com/ValentinDumas/website-status-checker/internal/monitor"
)

// mockNotifier records notifications for testing.
type mockNotifier struct {
	calls []mockCall
}

type mockCall struct {
	title   string
	message string
}

func (m *mockNotifier) Send(title, message string) error {
	m.calls = append(m.calls, mockCall{title: title, message: message})
	return nil
}

// ---------------------------------------------------------------------------
// StatusChangeHandler
// ---------------------------------------------------------------------------

func TestStatusChangeHandler_SiteDown(t *testing.T) {
	mock := &mockNotifier{}
	handler := StatusChangeHandler(mock)

	handler(monitor.SiteStatus{
		Site: config.Site{Name: "My Site"},
		LatestResult: checker.Result{
			IsUp:       false,
			StatusCode: 500,
			CheckedAt:  time.Now(),
		},
	})

	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(mock.calls))
	}
	if mock.calls[0].title != "🔴 Site Down" {
		t.Errorf("title = %q, want %q", mock.calls[0].title, "🔴 Site Down")
	}
}

func TestStatusChangeHandler_SiteDown_WithError(t *testing.T) {
	mock := &mockNotifier{}
	handler := StatusChangeHandler(mock)

	handler(monitor.SiteStatus{
		Site: config.Site{Name: "My Site"},
		LatestResult: checker.Result{
			IsUp:      false,
			Error:     fmt.Errorf("connection refused"),
			CheckedAt: time.Now(),
		},
	})

	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(mock.calls))
	}
	if !containsSubstring(mock.calls[0].message, "unreachable") {
		t.Errorf("message should mention 'unreachable', got: %s", mock.calls[0].message)
	}
}

func TestStatusChangeHandler_SiteRecovered(t *testing.T) {
	mock := &mockNotifier{}
	handler := StatusChangeHandler(mock)

	handler(monitor.SiteStatus{
		Site: config.Site{Name: "My Site"},
		LatestResult: checker.Result{
			IsUp:         true,
			StatusCode:   200,
			ResponseTime: 142 * time.Millisecond,
			CheckedAt:    time.Now(),
		},
	})

	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(mock.calls))
	}
	if mock.calls[0].title != "🟢 Site Recovered" {
		t.Errorf("title = %q, want %q", mock.calls[0].title, "🟢 Site Recovered")
	}
	if !containsSubstring(mock.calls[0].message, "142ms") {
		t.Errorf("message should contain response time, got: %s", mock.calls[0].message)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
