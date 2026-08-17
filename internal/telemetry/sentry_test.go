package telemetry

import (
	"errors"
	"testing"
)

// The no-DSN path must remain a complete no-op: no Sentry client, no flush
// work, and capture helpers must not panic.
func TestSentryDisabledWithoutDSN(t *testing.T) {
	sentryEnabled.Store(true)
	cleanup := ConfigureSentry(SentryConfig{DSN: ""})
	sentryEnabled.Store(false)
	cleanup()

	CaptureError(errors.New("ignored"), map[string]string{"operation": "test"})
	CapturePanic("ignored panic", []byte("stack"), map[string]string{"operation": "test"})
}
