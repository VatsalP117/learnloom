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
	"github.com/VatsalP117/learnloom/internal/failure"
	"github.com/VatsalP117/learnloom/internal/source"
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

func TestArtifactCleanupCompletesOnlyAfterObjectDeletion(t *testing.T) {
	t.Parallel()
	lifecycle := &artifactCleanupLifecycle{
		claim: &store.ArtifactCleanupClaim{
			Key: "accounts/a/issues/i/g.json.gz", Token: "cleanup-token",
			ExpiresAt: time.Now().Add(time.Minute),
		},
		completed: make(chan string, 1),
		failed:    make(chan error, 1),
	}
	artifacts := &artifactCleanupArtifacts{}
	worker, err := New(
		lifecycle, unusedProducer{}, artifacts, staticMailer{}, nil,
		validConfig(), slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err != nil {
		t.Fatal(err)
	}
	wantKey := lifecycle.claim.Key
	if err := worker.processArtifactCleanup(context.Background()); err != nil {
		t.Fatal(err)
	}
	if artifacts.deleted != wantKey {
		t.Fatalf("deleted key=%q", artifacts.deleted)
	}
	select {
	case key := <-lifecycle.completed:
		if key != wantKey {
			t.Fatalf("completed key=%q", key)
		}
	default:
		t.Fatal("artifact cleanup claim was not completed")
	}
	if worker.Snapshot().ArtifactCleanups != 1 {
		t.Fatal("artifact cleanup metric was not incremented")
	}
}

func TestArtifactCleanupFailureRemainsRetryable(t *testing.T) {
	t.Parallel()
	deleteErr := errors.New("object store unavailable")
	lifecycle := &artifactCleanupLifecycle{
		claim: &store.ArtifactCleanupClaim{
			Key: "accounts/a/issues/i/g.json.gz", Token: "cleanup-token",
			ExpiresAt: time.Now().Add(time.Minute),
		},
		completed: make(chan string, 1),
		failed:    make(chan error, 1),
	}
	worker, err := New(
		lifecycle, unusedProducer{}, &artifactCleanupArtifacts{deleteErr: deleteErr},
		staticMailer{}, nil, validConfig(),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := worker.processArtifactCleanup(context.Background()); !errors.Is(err, deleteErr) {
		t.Fatalf("cleanup err=%v", err)
	}
	select {
	case failed := <-lifecycle.failed:
		if !errors.Is(failed, deleteErr) {
			t.Fatalf("failed cause=%v", failed)
		}
	default:
		t.Fatal("artifact cleanup claim was not released for retry")
	}
	if worker.Snapshot().ArtifactCleanups != 0 {
		t.Fatal("failed cleanup incremented success metric")
	}
}

func TestWeeklyRecapRenderingUsesAbsoluteSafeLinks(t *testing.T) {
	t.Parallel()
	htmlBody, textBody := renderWeeklyRecap(store.WeeklyRecapPayload{
		LessonsCompleted: 2,
		Concepts:         []string{"Evidence < confidence"},
		Capabilities:     []string{"Can explain Evidence < confidence"},
		Connection:       "Connect evidence to calibration.",
		ActionLabel:      "Review now",
		ActionURL:        "/review",
	}, "https://app.learnloom.blog")
	if !strings.Contains(htmlBody, "Can explain Evidence &lt; confidence") ||
		!strings.Contains(htmlBody, `href="https://app.learnloom.blog/review"`) ||
		!strings.Contains(textBody, "https://app.learnloom.blog/review") {
		t.Fatalf("unsafe or incomplete Recap: html=%s text=%s", htmlBody, textBody)
	}
}

func TestInsufficientEvidenceDefersIssueWithoutFailingWorkerCycle(t *testing.T) {
	t.Parallel()
	deferred := make(chan error, 1)
	failed := make(chan error, 1)
	lifecycle := &issueTransitionLifecycle{deferred: deferred, failed: failed}
	preparationErr := failure.New(
		"no_worthwhile_evidence",
		failure.CategoryInsufficientEvidence,
		"source_intelligence",
		false,
		failure.PublicNoEvidence,
		errors.New("not enough independent evidence today"),
	)
	worker, err := New(
		lifecycle,
		unusedProducer{},
		staticArtifacts{},
		staticMailer{},
		staticSourcePreparer{err: preparationErr},
		validConfig(),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err != nil {
		t.Fatal(err)
	}
	claim := &store.IssueClaim{
		Issue: domain.Issue{
			ID: "issue-1", NewsletterID: "stream-1",
			Newsletter: domain.Newsletter{ID: "stream-1"},
		},
		AccountID: "account-1", Token: "claim-token",
		ExpiresAt: time.Now().Add(time.Minute),
	}

	if err := worker.processIssue(context.Background(), claim); err != nil {
		t.Fatalf("evidence deferral failed the worker cycle: %v", err)
	}
	select {
	case cause := <-deferred:
		if failure.Describe(cause).Category != failure.CategoryInsufficientEvidence {
			t.Fatalf("deferred cause=%v", cause)
		}
	default:
		t.Fatal("Issue was not deferred")
	}
	select {
	case cause := <-failed:
		t.Fatalf("Issue was incorrectly failed: %v", cause)
	default:
	}
	if worker.metrics.generationDeferred.Load() != 1 ||
		worker.metrics.generationFailed.Load() != 0 {
		t.Fatalf(
			"deferred=%d failed=%d",
			worker.metrics.generationDeferred.Load(),
			worker.metrics.generationFailed.Load(),
		)
	}
}

func TestEvidenceLedRhythmDefersUnchangedEvidenceBeforeModelGeneration(t *testing.T) {
	t.Parallel()
	deferred := make(chan error, 1)
	lifecycle := &issueTransitionLifecycle{
		deferred: deferred,
		failed:   make(chan error, 1),
	}
	worker, err := New(
		lifecycle,
		unusedProducer{},
		staticArtifacts{},
		staticMailer{},
		staticSourcePreparer{result: source.PrepareIssueResult{
			Items:            []domain.SourceItem{{SourceID: "S1", Title: "Same evidence"}},
			HasNovelEvidence: false,
		}},
		validConfig(),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err != nil {
		t.Fatal(err)
	}
	claim := &store.IssueClaim{
		Issue: domain.Issue{
			ID: "issue-evidence-rhythm", NewsletterID: "stream-1",
			Newsletter: domain.Newsletter{
				ID: "stream-1", RhythmMode: domain.RhythmEvidenceLed,
			},
		},
		AccountID: "account-1", Token: "claim-token",
		ExpiresAt: time.Now().Add(time.Minute),
	}
	if err := worker.processIssue(context.Background(), claim); err != nil {
		t.Fatalf("evidence-led deferral failed worker cycle: %v", err)
	}
	select {
	case cause := <-deferred:
		detail := failure.Describe(cause)
		if detail.Code != "no_new_evidence" || detail.Stage != "evidence_rhythm" {
			t.Fatalf("detail=%#v", detail)
		}
	default:
		t.Fatal("unchanged evidence was not deferred")
	}
}

func TestReviewModePausesAfterSourcePreparationBeforeModelGeneration(t *testing.T) {
	t.Parallel()
	awaiting := make(chan struct{}, 1)
	lifecycle := &issueTransitionLifecycle{
		awaiting: awaiting,
		deferred: make(chan error, 1),
		failed:   make(chan error, 1),
	}
	worker, err := New(
		lifecycle,
		unusedProducer{},
		staticArtifacts{},
		staticMailer{},
		staticSourcePreparer{},
		validConfig(),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err != nil {
		t.Fatal(err)
	}
	claim := &store.IssueClaim{
		Issue: domain.Issue{
			ID: "issue-review", NewsletterID: "stream-review",
			Newsletter: domain.Newsletter{
				ID: "stream-review", SourceReviewMode: domain.SourceReviewBeforeLesson,
			},
		},
		AccountID: "account-1", Token: "claim-token",
		ExpiresAt: time.Now().Add(time.Minute),
	}

	if err := worker.processIssue(context.Background(), claim); err != nil {
		t.Fatalf("approval pause failed the worker cycle: %v", err)
	}
	select {
	case <-awaiting:
	default:
		t.Fatal("Issue was not moved to source approval")
	}
	if worker.Snapshot().GenerationAwaitingApproval != 1 || worker.Snapshot().Generated != 0 {
		t.Fatalf("snapshot=%#v", worker.Snapshot())
	}
}

func TestModelTruncationIsCountedForAlerting(t *testing.T) {
	t.Parallel()
	failed := make(chan error, 1)
	lifecycle := &issueTransitionLifecycle{
		deferred: make(chan error, 1),
		failed:   failed,
	}
	truncation := failure.New(
		"model_output_truncated",
		failure.CategoryContentQuality,
		"editor",
		true,
		failure.PublicInternal,
		dossier.ErrOutputTruncated,
	)
	worker, err := New(
		lifecycle,
		unusedProducer{},
		staticArtifacts{},
		staticMailer{},
		staticSourcePreparer{err: truncation},
		validConfig(),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err != nil {
		t.Fatal(err)
	}
	claim := &store.IssueClaim{
		Issue: domain.Issue{
			ID: "issue-truncated", NewsletterID: "stream-1",
			Newsletter: domain.Newsletter{ID: "stream-1"},
		},
		AccountID: "account-1", Token: "claim-token",
		ExpiresAt: time.Now().Add(time.Minute),
	}
	if err := worker.processIssue(context.Background(), claim); err == nil {
		t.Fatal("truncation should remain observable to the worker cycle")
	}
	if worker.Snapshot().GenerationTruncated != 1 {
		t.Fatalf("truncation count=%d", worker.Snapshot().GenerationTruncated)
	}
	select {
	case cause := <-failed:
		if failure.Describe(cause).Code != "model_output_truncated" {
			t.Fatalf("failed cause=%v", cause)
		}
	default:
		t.Fatal("truncated Issue was not transitioned through durable retry")
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

type issueTransitionLifecycle struct {
	Lifecycle
	awaiting chan struct{}
	deferred chan error
	failed   chan error
}

func (l *issueTransitionLifecycle) AwaitSourceApproval(
	context.Context,
	string,
	string,
	time.Time,
) error {
	if l.awaiting != nil {
		l.awaiting <- struct{}{}
	}
	return nil
}

func (l *issueTransitionLifecycle) LoadLearningHistory(
	context.Context,
	string,
	int,
) ([]domain.LearningHistoryEntry, error) {
	return nil, nil
}

func (l *issueTransitionLifecycle) LoadLearnerState(
	context.Context,
	string,
	string,
	int,
) (domain.LearnerState, error) {
	return domain.LearnerState{}, nil
}

func (l *issueTransitionLifecycle) RenewIssueClaim(
	context.Context,
	string,
	string,
	time.Time,
) error {
	return nil
}

func (l *issueTransitionLifecycle) DeferIssue(
	_ context.Context,
	_, _ string,
	cause error,
	_ time.Time,
) error {
	l.deferred <- cause
	return nil
}

func (l *issueTransitionLifecycle) FailIssue(
	_ context.Context,
	_, _ string,
	cause error,
	_ int,
	_ time.Time,
) error {
	l.failed <- cause
	return nil
}

type staticSourcePreparer struct {
	result source.PrepareIssueResult
	err    error
}

func (s staticSourcePreparer) PrepareIssue(
	context.Context,
	domain.Newsletter,
	string,
) (source.PrepareIssueResult, error) {
	return s.result, s.err
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

type artifactCleanupLifecycle struct {
	Lifecycle
	claim     *store.ArtifactCleanupClaim
	completed chan string
	failed    chan error
}

func (l *artifactCleanupLifecycle) ClaimArtifactCleanup(
	context.Context,
	time.Time,
	time.Duration,
) (*store.ArtifactCleanupClaim, error) {
	claim := l.claim
	l.claim = nil
	return claim, nil
}

func (l *artifactCleanupLifecycle) CompleteArtifactCleanup(
	_ context.Context,
	key, _ string,
) error {
	l.completed <- key
	return nil
}

func (l *artifactCleanupLifecycle) FailArtifactCleanup(
	_ context.Context,
	_, _ string,
	cause error,
	_ time.Time,
) error {
	l.failed <- cause
	return nil
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

type artifactCleanupArtifacts struct {
	staticArtifacts
	deleted   string
	deleteErr error
}

func (a *artifactCleanupArtifacts) Delete(_ context.Context, key string) error {
	a.deleted = key
	return a.deleteErr
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
