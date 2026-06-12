package instagram

import (
	"testing"
	"time"
)

// SetContainerPollWaitForTest shortens container polling for tests.
func SetContainerPollWaitForTest(t *testing.T, d time.Duration) {
	t.Helper()
	old := containerPollWait
	containerPollWait = d
	t.Cleanup(func() { containerPollWait = old })
}
