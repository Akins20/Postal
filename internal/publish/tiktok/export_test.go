package tiktok

import (
	"testing"
	"time"
)

// SetStatusPollWaitForTest shortens publish-status polling for tests.
func SetStatusPollWaitForTest(t *testing.T, d time.Duration) {
	t.Helper()
	old := statusPollWait
	statusPollWait = d
	t.Cleanup(func() { statusPollWait = old })
}
