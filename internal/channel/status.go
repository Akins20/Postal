package channel

// Channel lifecycle statuses.
const (
	// statusActive means the channel has valid credentials and can publish.
	statusActive = "active"
	// statusExpired means token refresh failed; the user must reconnect.
	statusExpired = "expired"
)
