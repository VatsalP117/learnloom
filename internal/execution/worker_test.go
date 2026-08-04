package execution

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/VatsalP117/learnloom/internal/artifact"
	"github.com/VatsalP117/learnloom/internal/delivery"
	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/VatsalP117/learnloom/internal/dossier"
	"github.com/VatsalP117/learnloom/internal/store"
	"github.com/VatsalP117/learnloom/internal/telemetry"
)

func TestWorkerDurationMetricsRecordOutcome(t *testing.T) {
	worker := &Worker{metrics: workerMetrics{
		durations: telemetry.NewHistogramFamily([]float64{1, 5}),
	}}
	worker.observeDuration("issue_total", 2*time.Second, nil)
	worker.observeDuration("issue_total", 3*time.Second, errors.New("failed"))

	var output strings.Builder
	if err := worker.WriteDurationMetrics(&output); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		`learnloom_worker_operation_duration_seconds_count{operation="issue_total",outcome="success"} 1`,
		`learnloom_worker_operation_duration_seconds_count{operation="issue_total",outcome="failure"} 1`,
	} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("missing %q in:\n%s", expected, output.String())
		}
	}
}

func TestAcceptedDeliveryWithReceiptFailureBecomesUnknown(t *testing.T) {
	receiptFailure := errors.New("database unavailable after provider accepted email")
	lifecycle := &deliveryLifecycle{
		completeErr: receiptFailure,
		unknown:     make(chan error, 1),
	}
	worker, err := New(
		lifecycle,
		unusedProducer{},
		staticArtifacts{},
		staticMailer{},
		nil,
		validConfig(),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err != nil {
		t.Fatal(err)
	}
	claim := &store.DeliveryClaim{
		Token: "claim-token", PrimaryEmail: "learner@example.com",
		Issue: domain.Issue{
			ID: "issue-1", GenerationID: "generation-1",
			ArtifactKey: "accounts/a/issues/i/g.json", Title: "A Dossier",
		},
	}

	if err := worker.processDelivery(context.Background(), claim); err == nil {
		t.Fatal("expected the receipt failure to remain observable")
	}
	select {
	case marked := <-lifecycle.unknown:
		var unknown *delivery.OutcomeUnknownError
		if !errors.As(marked, &unknown) || !errors.Is(marked, receiptFailure) {
			t.Fatalf("unexpected transition cause: %v", marked)
		}
	default:
		t.Fatal("delivery was not moved to unknown")
	}
}

func TestInterruptedDeliveryBecomesUnknown(t *testing.T) {
	t.Parallel()
	lifecycle := &deliveryLifecycle{unknown: make(chan error, 1)}
	worker, err := New(
		lifecycle,
		unusedProducer{},
		staticArtifacts{},
		cancelledMailer{},
		nil,
		validConfig(),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err != nil {
		t.Fatal(err)
	}
	claim := &store.DeliveryClaim{
		Token: "claim-token", PrimaryEmail: "learner@example.com",
		ExpiresAt: time.Now().Add(time.Minute),
		Issue: domain.Issue{
			ID: "issue-1", GenerationID: "generation-1",
			ArtifactKey: "accounts/a/issues/i/g.json", Title: "A Dossier",
		},
	}
	if err := worker.processDelivery(context.Background(), claim); err == nil {
		t.Fatal("expected interrupted delivery to remain observable")
	}
	select {
	case marked := <-lifecycle.unknown:
		if !errors.Is(marked, context.Canceled) {
			t.Fatalf("unexpected transition cause: %v", marked)
		}
	default:
		t.Fatal("interrupted delivery was not moved to unknown")
	}
}

func TestWeeklyRecapRenderingUsesAbsoluteSafeLinks(t *testing.T) {
	t.Parallel()
	htmlBody, textBody := renderWeeklyRecap(store.WeeklyRecapPayload{
		LessonsCompleted: 2,
		Concepts:         []string{"Evidence < confidence"},
		Connection:       "Connect evidence to calibration.",
		ActionLabel:      "Review now",
		ActionURL:        "/review",
	}, "https://app.learnloom.blog")
	if !strings.Contains(htmlBody, "Evidence &lt; confidence") ||
		!strings.Contains(htmlBody, `href="https://app.learnloom.blog/review"`) ||
		!strings.Contains(textBody, "https://app.learnloom.blog/review") {
		t.Fatalf("unsafe or incomplete Recap: html=%s text=%s", htmlBody, textBody)
	}
}

func TestIssueClaimRenewalToleratesTransientDatabaseFailure(t *testing.T) {
	t.Parallel()
	lifecycle := &renewalLifecycle{}
	worker := &Worker{
		lifecycle: lifecycle,
		cfg:       Config{ClaimDuration: 30 * time.Millisecond},
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		now:       func() time.Time { return time.Now().UTC() },
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	claim := &store.IssueClaim{
		Issue: domain.Issue{ID: "issue-1"},
		Token: "claim-token", ExpiresAt: time.Now().Add(time.Second),
	}
	go worker.renewIssueClaim(ctx, claim, cancel, result)
	deadline := time.After(time.Second)
	for lifecycle.calls.Load() < 2 {
		select {
		case <-deadline:
			t.Fatal("Claim renewal was not retried")
		case <-time.After(5 * time.Millisecond):
		}
	}
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("renew result=%v", err)
	}
	if worker.metrics.renewalFailures.Load() != 1 {
		t.Fatalf("renewal failures=%d", worker.metrics.renewalFailures.Load())
	}
}

type renewalLifecycle struct {
	Lifecycle
	calls atomic.Int32
}

func (l *renewalLifecycle) RenewIssueClaim(
	context.Context,
	string,
	string,
	time.Time,
) error {
	if l.calls.Add(1) == 1 {
		return errors.New("temporary database interruption")
	}
	return nil
}

type deliveryLifecycle struct {
	Lifecycle
	completeErr error
	unknown     chan error
}

func (l *deliveryLifecycle) CompleteDelivery(
	context.Context,
	string,
	string,
	string,
	time.Time,
) error {
	return l.completeErr
}

func (l *deliveryLifecycle) MarkDeliveryUnknown(
	_ context.Context,
	_, _ string,
	cause error,
	_ time.Time,
) error {
	l.unknown <- cause
	return nil
}

func (l *deliveryLifecycle) RenewDeliveryClaim(
	context.Context,
	string,
	string,
	time.Time,
) error {
	return nil
}

type unusedProducer struct{}

func (unusedProducer) Generate(
	context.Context,
	dossier.GenerateRequest,
) (dossier.GenerateResult, error) {
	panic("unexpected generation")
}

type staticArtifacts struct{}

func (staticArtifacts) Put(
	context.Context,
	artifact.PutInput,
) (artifact.PutResult, error) {
	panic("unexpected artifact write")
}

func (staticArtifacts) Get(
	context.Context,
	string,
) (domain.DossierArtifact, error) {
	return domain.DossierArtifact{
		Dossier:  domain.Dossier{Version: 1, Title: "A Dossier"},
		Markdown: "# A Dossier",
		HTML:     "<h1>A Dossier</h1>",
	}, nil
}

func (staticArtifacts) Delete(context.Context, string) error { return nil }
func (staticArtifacts) DeleteAccount(context.Context, string) error {
	return nil
}

type staticMailer struct{}

func (staticMailer) Deliver(
	context.Context,
	delivery.Message,
) (string, error) {
	return "provider-email-id", nil
}

type cancelledMailer struct{}

func (cancelledMailer) Deliver(context.Context, delivery.Message) (string, error) {
	return "", context.Canceled
}

func validConfig() Config {
	return Config{
		PollInterval: time.Second, ClaimDuration: time.Minute,
		MaxIssueAttempts: 3, MaxDeliveryAttempts: 3,
		AccountConcurrency: 1, GlobalConcurrency: 1,
		DailyAccountLimit: 5, DailyGlobalLimit: 100,
		HistoryEntries: 10, RootDomain: "learnloom.blog",
	}
}
