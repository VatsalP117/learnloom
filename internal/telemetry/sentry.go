package telemetry

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/getsentry/sentry-go"
)

var sentryEnabled atomic.Bool

// SentryConfig mirrors the optional runtime Sentry settings loaded from
// configuration so the package stays decoupled from internal/config.
type SentryConfig struct {
	DSN         string
	Environment string
	Release     string
}

// ConfigureSentry initializes the process-wide Sentry client. It is a no-op
// when DSN is empty, so local development and CI run with zero Sentry
// overhead. The returned function flushes buffered events and should be
// deferred by the caller on process shutdown.
func ConfigureSentry(cfg SentryConfig) func() {
	if cfg.DSN == "" {
		return func() {}
	}
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		Release:          cfg.Release,
		SampleRate:       1.0,
		AttachStacktrace: true,
	}); err != nil {
		return func() {}
	}
	sentryEnabled.Store(true)
	return func() {
		sentry.Flush(5 * time.Second)
		sentryEnabled.Store(false)
	}
}

// CaptureError sends an error to Sentry with the given tags when Sentry is
// configured. It is a fast no-op otherwise. The returned value is ignored by
// callers; capture is best-effort and never blocks request handling.
func CaptureError(err error, tags map[string]string) {
	if err == nil || !sentryEnabled.Load() {
		return
	}
	sentry.WithScope(func(scope *sentry.Scope) {
		for key, value := range tags {
			scope.SetTag(key, value)
		}
		sentry.CaptureException(err)
	})
}

// CapturePanic reports a recovered panic to Sentry. The raw stack from the
// recovery point is attached as an extra field so the failing frames are
// visible in the Sentry event even though the panicking stack has unwound.
func CapturePanic(value any, stack []byte, tags map[string]string) {
	if value == nil || !sentryEnabled.Load() {
		return
	}
	sentry.WithScope(func(scope *sentry.Scope) {
		for key, value := range tags {
			scope.SetTag(key, value)
		}
		if len(stack) > 0 {
			scope.SetContext("panic", map[string]interface{}{
				"stack_trace": string(stack),
			})
		}
		sentry.CaptureMessage(fmt.Sprintf("panic: %v", value))
	})
}
