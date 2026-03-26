// Package notify sends desktop notifications when a site's status changes.
// It abstracts the notification backend behind an interface for testability
// and future cross-platform support.
package notify

import (
	"fmt"

	"github.com/gen2brain/beeep"

	"github.com/ValentinDumas/website-status-checker/internal/monitor"
)

// Notifier sends desktop notifications.
type Notifier interface {
	// Send displays a desktop notification with the given title and message.
	Send(title, message string) error
}

// DesktopNotifier sends native desktop notifications using the OS notification system.
// On Windows 10/11: toast notifications. On macOS: Notification Center. On Linux: notify-send.
type DesktopNotifier struct{}

// NewDesktopNotifier creates a new DesktopNotifier.
func NewDesktopNotifier() *DesktopNotifier {
	return &DesktopNotifier{}
}

// Send displays a desktop notification.
func (n *DesktopNotifier) Send(title, message string) error {
	return beeep.Notify(title, message, "")
}

// StatusChangeHandler returns a monitor.StatusChangeCallback that sends
// desktop notifications when a site goes down or recovers.
func StatusChangeHandler(notifier Notifier) monitor.StatusChangeCallback {
	return func(status monitor.SiteStatus) {
		var title, message string

		if status.LatestResult.IsUp {
			title = "🟢 Site Recovered"
			ms := status.LatestResult.ResponseTime.Milliseconds()
			message = fmt.Sprintf("%s is back up (%dms)", status.Site.Name, ms)
		} else {
			title = "🔴 Site Down"
			if status.LatestResult.Error != nil {
				message = fmt.Sprintf("%s is unreachable: %v", status.Site.Name, status.LatestResult.Error)
			} else {
				message = fmt.Sprintf("%s returned HTTP %d", status.Site.Name, status.LatestResult.StatusCode)
			}
		}

		// Best-effort: don't crash if notification fails.
		_ = notifier.Send(title, message)
	}
}
