package execution

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/VatsalP117/learnloom/internal/artifact"
	"github.com/VatsalP117/learnloom/internal/delivery"
	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/VatsalP117/learnloom/internal/dossier"
	"github.com/VatsalP117/learnloom/internal/failure"
	"github.com/VatsalP117/learnloom/internal/source"
	"github.com/VatsalP117/learnloom/internal/store"
	"github.com/VatsalP117/learnloom/internal/telemetry"
	"github.com/google/uuid"
)

type Lifecycle interface {
	RecoverExpiredClaims(context.Context, time.Time, int, int) (int64, error)
	DispatchDue(context.Context, time.Time, int) (int, error)
	ClaimNextIssue(context.Context, time.Time, time.Duration, int, int, int, int64, int64, store.IssueAttemptContext) (*store.IssueClaim, error)
	RenewIssueClaim(context.Context, string, string, time.Time) error
	ReleaseIssueClaim(context.Context, string, string, error, time.Time) error
	RecordIssueStage(context.Context, string, string, string, time.Duration, store.StageUsage, error, time.Time) error
	LoadIssueCheckpoints(context.Context, string, string) (map[string]string, error)
	SaveIssueCheckpoint(context.Context, string, string, string, string, string, string, time.Time) error
	LoadLearningHistory(context.Context, string, int) ([]domain.LearningHistoryEntry, error)
	LoadLearnerState(context.Context, string, string, int) (domain.LearnerState, error)
	CompleteIssue(context.Context, string, store.CompleteIssueInput) error
	FailIssue(context.Context, string, string, error, int, time.Time) error
	DeferIssue(context.Context, string, string, error, time.Time) error
	AwaitSourceApproval(context.Context, string, string, time.Time) error
	DispatchWeeklyRecaps(context.Context, time.Time, int) (int, error)
	ClaimNextWeeklyRecap(context.Context, time.Time, time.Duration, int) (*store.WeeklyRecapClaim, error)
	RenewWeeklyRecapClaim(context.Context, string, string, time.Time) error
	CompleteWeeklyRecap(context.Context, string, string, string, time.Time) error
	FailWeeklyRecap(context.Context, string, string, error, int, time.Time) error
	MarkWeeklyRecapUnknown(context.Context, string, string, error, time.Time) error
	DispatchPublicFollowUpdates(context.Context, time.Time, int) (int, error)
	ClaimNextPublicFollowDelivery(context.Context, time.Time, time.Duration, int) (*store.PublicFollowClaim, error)
	RenewPublicFollowDeliveryClaim(context.Context, string, string, time.Time) error
	CompletePublicFollowDelivery(context.Context, string, string, string, time.Time) error
	FailPublicFollowDelivery(context.Context, string, string, error, int, time.Time) error
	MarkPublicFollowDeliveryUnknown(context.Context, string, string, error, time.Time) error
	ClaimNextDelivery(context.Context, time.Time, time.Duration, int) (*store.DeliveryClaim, error)
	RenewDeliveryClaim(context.Context, string, string, time.Time) error
	CompleteDelivery(context.Context, string, string, string, time.Time) error
	FailDelivery(context.Context, string, string, error, int, time.Time) error
	MarkDeliveryUnknown(context.Context, string, string, error, time.Time) error
	ClaimAccountDeletion(context.Context, time.Time, time.Duration) (*store.DeletionClaim, error)
	CompleteAccountDeletion(context.Context, string, string, time.Time) error
	FailAccountDeletion(context.Context, string, string, error, time.Time) error
	RegisterArtifactCleanup(context.Context, string, string, time.Time) error
	ClaimArtifactCleanup(context.Context, time.Time, time.Duration) (*store.ArtifactCleanupClaim, error)
	CompleteArtifactCleanup(context.Context, string, string) error
	CancelArtifactCleanup(context.Context, string) error
	FailArtifactCleanup(context.Context, string, string, error, time.Time) error
	CleanupOperationalState(context.Context, time.Time) (int64, error)
}

type Producer interface {
	Generate(context.Context, dossier.GenerateRequest) (dossier.GenerateResult, error)
}

type SourcePreparer interface {
	PrepareIssue(
		context.Context,
		domain.Newsletter,
		string,
	) (source.PrepareIssueResult, error)
}

type Artifacts interface {
	Put(context.Context, artifact.PutInput) (artifact.PutResult, error)
	Get(context.Context, string) (domain.DossierArtifact, error)
	Delete(context.Context, string) error
	DeleteAccount(context.Context, string) error
}

type Mailer interface {
	Deliver(context.Context, delivery.Message) (string, error)
}

type Config struct {
	PollInterval             time.Duration
	ClaimDuration            time.Duration
	MaxIssueAttempts         int
	MaxDeliveryAttempts      int
	AccountConcurrency       int
	GlobalConcurrency        int
	IssueTimeout             time.Duration
	DailyAccountLimit        int
	DailyGlobalLimit         int
	DailyModelBudgetMicroUSD int64
	ModelReservationMicroUSD int64
	HistoryEntries           int
	RootDomain               string
	AppOrigin                string
	AttemptContext           store.IssueAttemptContext
}

type Worker struct {
	lifecycle   Lifecycle
	producer    Producer
	artifacts   Artifacts
	mailer      Mailer
	sourceSvc   SourcePreparer
	cfg         Config
	logger      *slog.Logger
	now         func() time.Time
	lastCleanup time.Time
	metrics     workerMetrics
	draining    atomic.Bool
}

type workerMetrics struct {
	cycles                     atomic.Uint64
	generated                  atomic.Uint64
	generationFailed           atomic.Uint64
	generationDeferred         atomic.Uint64
	generationTruncated        atomic.Uint64
	generationAwaitingApproval atomic.Uint64
	delivered                  atomic.Uint64
	deliveryFailed             atomic.Uint64
	deletions                  atomic.Uint64
	artifactCleanups           atomic.Uint64
	recoveredClaims            atomic.Uint64
	renewalFailures            atomic.Uint64
	releasedClaims             atomic.Uint64
	activeIssues               atomic.Int64
	activeDeliveries           atomic.Int64
	lastCycleUnixNano          atomic.Int64
	durations                  *telemetry.HistogramFamily
}

type Snapshot struct {
	Cycles                     uint64    `json:"cycles"`
	Generated                  uint64    `json:"generated"`
	GenerationFailed           uint64    `json:"generationFailed"`
	GenerationDeferred         uint64    `json:"generationDeferred"`
	GenerationTruncated        uint64    `json:"generationTruncated"`
	GenerationAwaitingApproval uint64    `json:"generationAwaitingApproval"`
	Delivered                  uint64    `json:"delivered"`
	DeliveryFailed             uint64    `json:"deliveryFailed"`
	Deletions                  uint64    `json:"deletions"`
	ArtifactCleanups           uint64    `json:"artifactCleanups"`
	RecoveredClaims            uint64    `json:"recoveredClaims"`
	RenewalFailures            uint64    `json:"renewalFailures"`
	ReleasedClaims             uint64    `json:"releasedClaims"`
	ActiveIssues               int64     `json:"activeIssues"`
	ActiveDeliveries           int64     `json:"activeDeliveries"`
	Draining                   bool      `json:"draining"`
	LastCycleAt                time.Time `json:"lastCycleAt"`
}

func New(
	lifecycle Lifecycle,
	producer Producer,
	artifacts Artifacts,
	mailer Mailer,
	sourceSvc SourcePreparer,
	cfg Config,
	logger *slog.Logger,
) (*Worker, error) {
	if lifecycle == nil || producer == nil || artifacts == nil || mailer == nil {
		return nil, errors.New("Issue execution dependencies are required")
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 2 * time.Second
	}
	if cfg.IssueTimeout == 0 {
		cfg.IssueTimeout = 45 * time.Minute
	}
	if cfg.ClaimDuration < time.Minute {
		return nil, errors.New("Issue Claim duration must be at least one minute")
	}
	if cfg.MaxIssueAttempts < 1 || cfg.MaxDeliveryAttempts < 1 ||
		cfg.AccountConcurrency < 1 || cfg.GlobalConcurrency < 1 ||
		cfg.DailyAccountLimit < 1 || cfg.DailyGlobalLimit < 1 {
		return nil, errors.New("Issue execution limits are invalid")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{
		lifecycle: lifecycle, producer: producer, artifacts: artifacts,
		mailer: mailer, sourceSvc: sourceSvc, cfg: cfg, logger: logger,
		now: func() time.Time { return time.Now().UTC() },
		metrics: workerMetrics{
			durations: telemetry.NewHistogramFamily([]float64{
				0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30,
				60, 120, 300, 600, 1200, 1800, 2700,
			}),
		},
	}, nil
}

func (w *Worker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()
	for {
		if w.draining.Load() {
			return nil
		}
		if err := w.Cycle(ctx); err != nil && !errors.Is(err, context.Canceled) {
			w.logger.ErrorContext(ctx, "worker cycle failed", "error", err)
		}
		if w.draining.Load() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (w *Worker) BeginDrain() {
	w.draining.Store(true)
}

func (w *Worker) Cycle(ctx context.Context) error {
	now := w.now()
	w.metrics.cycles.Add(1)
	w.metrics.lastCycleUnixNano.Store(now.UnixNano())
	recovered, err := w.lifecycle.RecoverExpiredClaims(
		ctx,
		now,
		w.cfg.MaxIssueAttempts,
		w.cfg.MaxDeliveryAttempts,
	)
	if err != nil {
		w.captureError(err, "recover_expired_claims", nil)
		return err
	}
	if recovered > 0 {
		w.metrics.recoveredClaims.Add(uint64(recovered))
		w.logger.WarnContext(ctx, "recovered expired claims", "count", recovered)
	}
	if w.draining.Load() {
		return nil
	}
	dispatched, err := w.lifecycle.DispatchDue(ctx, now, 100)
	if err != nil {
		w.captureError(err, "dispatch_issues", nil)
		return err
	}
	if dispatched > 0 {
		w.logger.InfoContext(ctx, "dispatched scheduled Issues", "count", dispatched)
	}
	if err := w.processIssues(ctx); err != nil {
		return err
	}
	if w.draining.Load() {
		return nil
	}
	if err := w.processDeliveries(ctx); err != nil {
		return err
	}
	if dispatched, err := w.lifecycle.DispatchWeeklyRecaps(ctx, now, 100); err != nil {
		w.captureError(err, "dispatch_weekly_recaps", nil)
		return err
	} else if dispatched > 0 {
		w.logger.InfoContext(ctx, "dispatched weekly Recaps", "count", dispatched)
	}
	if err := w.processWeeklyRecaps(ctx); err != nil {
		return err
	}
	if dispatched, err := w.lifecycle.DispatchPublicFollowUpdates(ctx, now, 100); err != nil {
		w.captureError(err, "dispatch_public_follow", nil)
		return err
	} else if dispatched > 0 {
		w.logger.InfoContext(ctx, "dispatched public path updates", "count", dispatched)
	}
	if err := w.processPublicFollowDeliveries(ctx); err != nil {
		return err
	}
	if w.draining.Load() {
		return nil
	}
	if err := w.processDeletion(ctx); err != nil {
		return err
	}
	if err := w.processArtifactCleanup(ctx); err != nil {
		return err
	}
	if w.lastCleanup.IsZero() || now.Sub(w.lastCleanup) >= time.Hour {
		if _, err := w.lifecycle.CleanupOperationalState(
			ctx,
			now.Add(-30*24*time.Hour),
		); err != nil {
			w.captureError(err, "operational_cleanup", nil)
			return err
		}
		w.lastCleanup = now
	}
	return nil
}

func (w *Worker) processPublicFollowDeliveries(ctx context.Context) error {
	for count := 0; count < w.cfg.GlobalConcurrency; count++ {
		if w.draining.Load() {
			return nil
		}
		claim, err := w.lifecycle.ClaimNextPublicFollowDelivery(
			ctx, w.now(), w.cfg.ClaimDuration, w.cfg.MaxDeliveryAttempts,
		)
		if err != nil {
			w.captureError(err, "claim_next_public_follow", nil)
			return err
		}
		if claim == nil {
			return nil
		}
		if err := w.processPublicFollowDelivery(ctx, claim); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) processPublicFollowDelivery(
	ctx context.Context,
	claim *store.PublicFollowClaim,
) error {
	ctx, cancel := context.WithCancel(ctx)
	renewed := make(chan error, 1)
	go w.renewPublicFollowDeliveryClaim(ctx, claim, cancel, renewed)
	appOrigin := strings.TrimRight(w.cfg.AppOrigin, "/")
	subject := "Confirm your Learnloom path follow"
	var htmlBody, textBody string
	if claim.Kind == "update" {
		dossierURL := "https://" + claim.SiteUsername + "." + w.cfg.RootDomain + "/d/" +
			claim.IssuePublicID + "/" + claim.IssuePublicSlug
		unsubscribeURL := appOrigin + "/public-follow/unsubscribe?token=" + claim.Token
		subject = claim.IssueTitle
		htmlBody = `<p>A new Dossier was published in <strong>` + html.EscapeString(claim.NewsletterName) + `</strong>.</p>` +
			`<p><a href="` + html.EscapeString(dossierURL) + `">` + html.EscapeString(claim.IssueTitle) + `</a></p>` +
			`<p><a href="` + html.EscapeString(unsubscribeURL) + `">Unsubscribe from this path</a></p>`
		textBody = "A new Dossier was published in " + claim.NewsletterName + ":\n\n" +
			claim.IssueTitle + "\n" + dossierURL + "\n\nUnsubscribe: " + unsubscribeURL + "\n"
	} else {
		confirmationURL := appOrigin + "/public-follow/confirm?token=" + claim.Token
		htmlBody = `<p>Confirm that you want to follow <strong>` + html.EscapeString(claim.NewsletterName) + `</strong>.</p>` +
			`<p><a href="` + html.EscapeString(confirmationURL) + `">Confirm this follow</a></p>` +
			`<p>If you did not request this, ignore this email.</p>`
		textBody = "Confirm that you want to follow " + claim.NewsletterName + ":\n\n" + confirmationURL +
			"\n\nIf you did not request this, ignore this email.\n"
	}
	externalID, err := w.mailer.Deliver(ctx, delivery.Message{
		IdempotencyKey: "public-follow/" + claim.ID,
		To:             claim.Email, Subject: subject,
		HTML: htmlBody, Text: textBody,
	})
	if err == nil {
		err = w.lifecycle.CompletePublicFollowDelivery(
			ctx, claim.ID, claim.ClaimToken, externalID, w.now(),
		)
		if err != nil {
			err = &delivery.OutcomeUnknownError{Cause: err}
		}
	}
	cancel()
	renewErr := <-renewed
	if err == nil && renewErr != nil && !errors.Is(renewErr, context.Canceled) {
		err = renewErr
	}
	if err == nil {
		w.metrics.delivered.Add(1)
		return nil
	}
	w.metrics.deliveryFailed.Add(1)
	w.captureError(err, "public_follow_delivery", map[string]string{
		"claim_id": claim.ID,
	})
	var unknown *delivery.OutcomeUnknownError
	var transitionErr error
	if errors.As(err, &unknown) || errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) {
		transitionErr = w.lifecycle.MarkPublicFollowDeliveryUnknown(
			context.Background(), claim.ID, claim.ClaimToken, err, w.now(),
		)
	} else {
		transitionErr = w.lifecycle.FailPublicFollowDelivery(
			context.Background(), claim.ID, claim.ClaimToken, err,
			w.cfg.MaxDeliveryAttempts, w.now(),
		)
	}
	if transitionErr != nil && !errors.Is(transitionErr, store.ErrClaimLost) {
		return errors.Join(err, transitionErr)
	}
	return fmt.Errorf("deliver public follow confirmation %s: %w", claim.ID, err)
}

func (w *Worker) renewPublicFollowDeliveryClaim(
	ctx context.Context,
	claim *store.PublicFollowClaim,
	cancel context.CancelFunc,
	result chan<- error,
) {
	interval := min(w.cfg.ClaimDuration/3, 30*time.Second)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	currentExpiry := claim.ExpiresAt
	for {
		select {
		case <-ctx.Done():
			result <- ctx.Err()
			return
		case <-ticker.C:
			expires := w.now().Add(w.cfg.ClaimDuration)
			err := w.lifecycle.RenewPublicFollowDeliveryClaim(
				ctx, claim.ID, claim.ClaimToken, expires,
			)
			if err != nil {
				w.metrics.renewalFailures.Add(1)
				if errors.Is(err, store.ErrClaimLost) || !w.now().Before(currentExpiry) {
					cancel()
					result <- err
					return
				}
				continue
			}
			currentExpiry = expires
		}
	}
}

func (w *Worker) processWeeklyRecaps(ctx context.Context) error {
	var wg sync.WaitGroup
	errorsChannel := make(chan error, w.cfg.GlobalConcurrency)
	for count := 0; count < w.cfg.GlobalConcurrency; count++ {
		if w.draining.Load() {
			break
		}
		claim, err := w.lifecycle.ClaimNextWeeklyRecap(
			ctx,
			w.now(),
			w.cfg.ClaimDuration,
			w.cfg.MaxDeliveryAttempts,
		)
		if err != nil {
			w.captureError(err, "claim_next_weekly_recap", nil)
			return err
		}
		if claim == nil {
			break
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := w.processWeeklyRecap(ctx, claim); err != nil {
				errorsChannel <- err
			}
		}()
	}
	wg.Wait()
	close(errorsChannel)
	return errors.Join(channelErrors(errorsChannel)...)
}

func (w *Worker) processWeeklyRecap(
	ctx context.Context,
	claim *store.WeeklyRecapClaim,
) error {
	ctx, cancel := context.WithCancel(ctx)
	renewed := make(chan error, 1)
	go w.renewWeeklyRecapClaim(ctx, claim, cancel, renewed)
	recapHTML, recapText := renderWeeklyRecap(claim.Payload, w.cfg.AppOrigin)
	externalID, err := w.mailer.Deliver(ctx, delivery.Message{
		IdempotencyKey: "recap/" + claim.ID + "/" + claim.WeekStart,
		To:             claim.PrimaryEmail,
		Subject:        "Your weekly learning recap",
		HTML:           recapHTML,
		Text:           recapText,
	})
	if err == nil {
		if completeErr := w.lifecycle.CompleteWeeklyRecap(
			ctx,
			claim.ID,
			claim.Token,
			externalID,
			w.now(),
		); completeErr != nil {
			err = &delivery.OutcomeUnknownError{Cause: completeErr}
		}
	}
	cancel()
	renewErr := <-renewed
	if err == nil && renewErr != nil && !errors.Is(renewErr, context.Canceled) {
		err = renewErr
	}
	if err == nil {
		w.metrics.delivered.Add(1)
		return nil
	}
	w.metrics.deliveryFailed.Add(1)
	w.captureError(err, "weekly_recap", map[string]string{
		"recap_id":   claim.ID,
		"week_start": claim.WeekStart,
	})
	var unknown *delivery.OutcomeUnknownError
	var transitionErr error
	if errors.As(err, &unknown) ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) {
		transitionErr = w.lifecycle.MarkWeeklyRecapUnknown(
			context.Background(),
			claim.ID,
			claim.Token,
			err,
			w.now(),
		)
	} else {
		transitionErr = w.lifecycle.FailWeeklyRecap(
			context.Background(),
			claim.ID,
			claim.Token,
			err,
			w.cfg.MaxDeliveryAttempts,
			w.now(),
		)
	}
	if transitionErr != nil && !errors.Is(transitionErr, store.ErrClaimLost) {
		return errors.Join(err, transitionErr)
	}
	return fmt.Errorf("deliver weekly Recap %s: %w", claim.ID, err)
}

func (w *Worker) renewWeeklyRecapClaim(
	ctx context.Context,
	claim *store.WeeklyRecapClaim,
	cancel context.CancelFunc,
	result chan<- error,
) {
	interval := min(w.cfg.ClaimDuration/3, 30*time.Second)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	currentExpiry := claim.ExpiresAt
	for {
		select {
		case <-ctx.Done():
			result <- ctx.Err()
			return
		case <-ticker.C:
			expires := w.now().Add(w.cfg.ClaimDuration)
			err := w.lifecycle.RenewWeeklyRecapClaim(
				ctx,
				claim.ID,
				claim.Token,
				expires,
			)
			if err != nil {
				w.metrics.renewalFailures.Add(1)
				if errors.Is(err, store.ErrClaimLost) ||
					!w.now().Before(currentExpiry) {
					cancel()
					result <- err
					return
				}
				continue
			}
			currentExpiry = expires
		}
	}
}

func renderWeeklyRecap(
	payload store.WeeklyRecapPayload,
	appOrigin string,
) (string, string) {
	capabilities := payload.Capabilities
	if len(capabilities) == 0 {
		for _, concept := range payload.Concepts {
			capabilities = append(capabilities, "Can explain the core idea behind "+concept)
		}
	}
	if len(capabilities) == 0 {
		capabilities = []string{"Your recent learning thread is ready to reconnect"}
	}
	var htmlBody strings.Builder
	var textBody strings.Builder
	actionURL := payload.ActionURL
	if strings.HasPrefix(actionURL, "/") && strings.TrimSpace(appOrigin) != "" {
		actionURL = strings.TrimRight(appOrigin, "/") + actionURL
	}
	fmt.Fprintf(
		&htmlBody,
		"<h1>Your week in learning</h1><p>You completed %d lesson%s.</p><h2>Capabilities gained</h2><ul>",
		payload.LessonsCompleted,
		map[bool]string{true: "", false: "s"}[payload.LessonsCompleted == 1],
	)
	fmt.Fprintf(
		&textBody,
		"Your week in learning\n\nYou completed %d lesson(s).\n\nCapabilities gained:\n",
		payload.LessonsCompleted,
	)
	for _, capability := range capabilities {
		fmt.Fprintf(&htmlBody, "<li>%s</li>", html.EscapeString(capability))
		fmt.Fprintf(&textBody, "- %s\n", capability)
	}
	fmt.Fprintf(
		&htmlBody,
		"</ul><h2>One useful connection</h2><p>%s</p>",
		html.EscapeString(payload.Connection),
	)
	fmt.Fprintf(&textBody, "\nOne useful connection:\n%s\n", payload.Connection)
	if payload.ReviewPrompt != "" {
		fmt.Fprintf(
			&htmlBody,
			"<h2>Try one retrieval</h2><p>%s</p>",
			html.EscapeString(payload.ReviewPrompt),
		)
		fmt.Fprintf(&textBody, "\nTry one retrieval:\n%s\n", payload.ReviewPrompt)
	}
	fmt.Fprintf(
		&htmlBody,
		`<p><a href="%s">%s</a></p>`,
		html.EscapeString(actionURL),
		html.EscapeString(payload.ActionLabel),
	)
	fmt.Fprintf(&textBody, "\n%s: %s\n", payload.ActionLabel, actionURL)
	return htmlBody.String(), textBody.String()
}

func (w *Worker) processIssues(ctx context.Context) error {
	var wg sync.WaitGroup
	errorsChannel := make(chan error, w.cfg.GlobalConcurrency)
	for count := 0; count < w.cfg.GlobalConcurrency; count++ {
		if w.draining.Load() {
			break
		}
		claim, err := w.lifecycle.ClaimNextIssue(
			ctx,
			w.now(),
			w.cfg.ClaimDuration,
			w.cfg.AccountConcurrency,
			w.cfg.DailyAccountLimit,
			w.cfg.DailyGlobalLimit,
			w.cfg.DailyModelBudgetMicroUSD,
			w.cfg.ModelReservationMicroUSD,
			w.cfg.AttemptContext,
		)
		if errors.Is(err, store.ErrGenerationPaused) ||
			errors.Is(err, store.ErrQuotaExceeded) {
			break
		}
		if err != nil {
			w.captureError(err, "claim_next_issue", nil)
			return err
		}
		if claim == nil {
			break
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.metrics.activeIssues.Add(1)
			defer w.metrics.activeIssues.Add(-1)
			if err := w.processIssue(ctx, claim); err != nil {
				errorsChannel <- err
			}
		}()
	}
	wg.Wait()
	close(errorsChannel)
	return errors.Join(channelErrors(errorsChannel)...)
}

func (w *Worker) processIssue(ctx context.Context, claim *store.IssueClaim) error {
	issueStarted := time.Now()
	ctx, cancel := context.WithTimeout(ctx, w.cfg.IssueTimeout)
	renewed := make(chan error, 1)
	go w.renewIssueClaim(ctx, claim, cancel, renewed)
	var err error
	awaitingSourceApproval := false
	phaseStarted := time.Now()
	history, err := w.lifecycle.LoadLearningHistory(
		ctx,
		claim.Issue.NewsletterID,
		w.cfg.HistoryEntries,
	)
	w.logIssuePhase(ctx, claim.Issue.ID, "history", phaseStarted, err)
	var learnerState domain.LearnerState
	if err == nil {
		phaseStarted = time.Now()
		learnerState, err = w.lifecycle.LoadLearnerState(
			ctx,
			claim.AccountID,
			claim.Issue.NewsletterID,
			w.cfg.HistoryEntries,
		)
		w.logIssuePhase(ctx, claim.Issue.ID, "learner_state", phaseStarted, err)
	}
	if err == nil {
		if w.sourceSvc == nil {
			err = errors.New("source intelligence service is unavailable")
		}
	}
	if err == nil {
		var prepared source.PrepareIssueResult
		phaseStarted = time.Now()
		prepared, err = w.sourceSvc.PrepareIssue(ctx, claim.Issue.Newsletter, claim.Issue.ID)
		w.logIssuePhase(ctx, claim.Issue.ID, "source_intelligence", phaseStarted, err)
		if err == nil &&
			claim.Issue.Newsletter.RhythmMode == domain.RhythmEvidenceLed &&
			!prepared.HasNovelEvidence {
			err = failure.New(
				failure.CodeNoNewEvidence,
				failure.CategoryInsufficientEvidence,
				"evidence_rhythm",
				false,
				failure.PublicNoEvidence,
				errors.New("evidence-led rhythm found no source changes since the previous lesson"),
			)
		}
		if err == nil &&
			claim.Issue.Newsletter.SourceReviewMode == domain.SourceReviewBeforeLesson &&
			claim.Issue.Newsletter.SourceApprovedAt == nil {
			awaitingSourceApproval = true
		}
		if err == nil && !awaitingSourceApproval {
			generateRequest := dossier.GenerateRequest{
				Newsletter:          claim.Issue.Newsletter,
				History:             history,
				LearnerState:        learnerState,
				Now:                 w.now(),
				RequestedLessonType: claim.Issue.RequestedLessonType,
				PreparedItems:       prepared.Items,
				Warnings:            prepared.Warnings,
			}
			fingerprint, fingerprintErr := dossier.GenerationFingerprint(
				generateRequest,
				w.cfg.AttemptContext.ModelName,
			)
			if fingerprintErr != nil {
				err = fingerprintErr
			}
			var checkpoints map[string]string
			if err == nil {
				checkpoints, err = w.lifecycle.LoadIssueCheckpoints(
					ctx,
					claim.Issue.ID,
					fingerprint,
				)
			}
			if err != nil {
				w.logIssuePhase(ctx, claim.Issue.ID, "checkpoint_load", phaseStarted, err)
			}
			var result dossier.GenerateResult
			phaseStarted = time.Now()
			if err == nil {
				generateRequest.Checkpoints = checkpoints
				generateRequest.OnCheckpoint = func(stage, output string) {
					checkpointCtx, checkpointCancel := context.WithTimeout(
						context.Background(),
						3*time.Second,
					)
					defer checkpointCancel()
					if checkpointErr := w.lifecycle.SaveIssueCheckpoint(
						checkpointCtx,
						claim.Issue.ID,
						claim.Token,
						fingerprint,
						stage,
						output,
						dossier.PipelineVersion,
						w.now(),
					); checkpointErr != nil && !errors.Is(checkpointErr, store.ErrClaimLost) {
						w.logger.WarnContext(
							ctx,
							"save Dossier checkpoint failed",
							"issue_id", claim.Issue.ID,
							"stage", stage,
							"error", checkpointErr,
						)
					}
				}
				generateRequest.OnStage = func(
					stage string,
					duration time.Duration,
					usage dossier.ModelUsage,
					stageErr error,
				) {
					w.observeDuration("model_stage_"+stage, duration, stageErr)
					w.logger.InfoContext(
						ctx,
						"Dossier model stage completed",
						"issue_id", claim.Issue.ID,
						"stage", stage,
						"duration_ms", duration.Milliseconds(),
						"success", stageErr == nil,
					)
					recordCtx, recordCancel := context.WithTimeout(
						context.Background(),
						3*time.Second,
					)
					defer recordCancel()
					if recordErr := w.lifecycle.RecordIssueStage(
						recordCtx,
						claim.Issue.ID,
						claim.Token,
						stage,
						duration,
						store.StageUsage{
							InputTokens:           usage.InputTokens,
							OutputTokens:          usage.OutputTokens,
							ProviderRetries:       usage.Retries,
							EstimatedCostMicroUSD: usage.EstimatedCostMicroUSD,
						},
						stageErr,
						w.now(),
					); recordErr != nil && !errors.Is(recordErr, store.ErrClaimLost) {
						w.logger.WarnContext(
							ctx,
							"record Dossier stage failed",
							"issue_id", claim.Issue.ID,
							"stage", stage,
							"error", recordErr,
						)
					}
				}
				result, err = w.producer.Generate(ctx, generateRequest)
			}
			w.logIssuePhase(ctx, claim.Issue.ID, "dossier_generation", phaseStarted, err)
			if err == nil {
				generationID := uuid.NewString()
				artifactKey, keyErr := artifact.KeyFor(
					claim.AccountID, claim.Issue.NewsletterID,
					claim.Issue.ID, generationID,
				)
				if keyErr != nil {
					err = keyErr
				} else {
					err = w.lifecycle.RegisterArtifactCleanup(
						ctx, artifactKey, claim.Issue.ID, w.now(),
					)
				}
				var saved artifact.PutResult
				if err == nil {
					phaseStarted = time.Now()
					saved, err = w.artifacts.Put(ctx, artifact.PutInput{
						AccountID: claim.AccountID, NewsletterID: claim.Issue.NewsletterID,
						IssueID: claim.Issue.ID, GenerationID: generationID,
						Artifact: result.Artifact,
					})
					w.logIssuePhase(ctx, claim.Issue.ID, "artifact_storage", phaseStarted, err)
				}
				if err == nil {
					phaseStarted = time.Now()
					err = w.lifecycle.CompleteIssue(ctx, claim.Issue.ID, store.CompleteIssueInput{
						ClaimToken: claim.Token, GenerationID: generationID,
						ArtifactKey: saved.Key, Checksum: saved.Checksum, Bytes: saved.Bytes,
						Title: result.Artifact.Dossier.Title, History: result.History,
						HistoryLimit: w.cfg.HistoryEntries, CompletedAt: w.now(),
					})
					w.logIssuePhase(ctx, claim.Issue.ID, "completion_transaction", phaseStarted, err)
					if err != nil {
						if deleteErr := w.artifacts.Delete(context.Background(), saved.Key); deleteErr == nil {
							_ = w.lifecycle.CancelArtifactCleanup(context.Background(), saved.Key)
						}
					}
				}
			}
		}
	}
	cancel()
	renewErr := <-renewed
	if awaitingSourceApproval {
		if renewErr != nil && !errors.Is(renewErr, context.Canceled) {
			return fmt.Errorf("await source approval for Issue %s: %w", claim.Issue.ID, renewErr)
		}
		if transitionErr := w.lifecycle.AwaitSourceApproval(
			context.Background(),
			claim.Issue.ID,
			claim.Token,
			w.now(),
		); transitionErr != nil {
			return fmt.Errorf("await source approval for Issue %s: %w", claim.Issue.ID, transitionErr)
		}
		w.metrics.generationAwaitingApproval.Add(1)
		w.logIssuePhase(ctx, claim.Issue.ID, "total", issueStarted, nil)
		return nil
	}
	if err == nil && renewErr != nil && !errors.Is(renewErr, context.Canceled) {
		err = renewErr
	}
	if err != nil {
		detail := failure.Describe(err)
		if detail.Code == "model_output_truncated" {
			w.metrics.generationTruncated.Add(1)
		}
		var failErr error
		if detail.Category == failure.CategoryInsufficientEvidence {
			failErr = w.lifecycle.DeferIssue(
				context.Background(),
				claim.Issue.ID,
				claim.Token,
				err,
				w.now(),
			)
			if failErr == nil {
				w.metrics.generationDeferred.Add(1)
				w.logIssuePhase(ctx, claim.Issue.ID, "total", issueStarted, err)
				return nil
			}
		} else if w.draining.Load() && errors.Is(err, context.Canceled) {
			failErr = w.lifecycle.ReleaseIssueClaim(
				context.Background(),
				claim.Issue.ID,
				claim.Token,
				err,
				w.now(),
			)
			if failErr == nil {
				w.metrics.releasedClaims.Add(1)
			}
		} else {
			w.metrics.generationFailed.Add(1)
			w.captureError(err, "generate_issue", map[string]string{
				"issue_id":         claim.Issue.ID,
				"incident_id":      detail.IncidentID,
				"failure_code":     detail.Code,
				"failure_category": string(detail.Category),
				"failure_stage":    detail.Stage,
				"retryable":        fmt.Sprintf("%t", detail.Retryable),
				"model":            w.cfg.AttemptContext.ModelName,
				"pipeline":         w.cfg.AttemptContext.PipelineVersion,
			})
			failErr = w.lifecycle.FailIssue(
				context.Background(),
				claim.Issue.ID,
				claim.Token,
				err,
				w.cfg.MaxIssueAttempts,
				w.now(),
			)
		}
		if failErr != nil && !errors.Is(failErr, store.ErrClaimLost) {
			err = errors.Join(err, failErr)
		}
		w.logIssuePhase(ctx, claim.Issue.ID, "total", issueStarted, err)
		return fmt.Errorf("generate Issue %s: %w", claim.Issue.ID, err)
	}
	w.metrics.generated.Add(1)
	w.logIssuePhase(ctx, claim.Issue.ID, "total", issueStarted, nil)
	return nil
}

func (w *Worker) logIssuePhase(
	ctx context.Context,
	issueID, phase string,
	started time.Time,
	err error,
) {
	duration := time.Since(started)
	w.observeDuration("issue_"+phase, duration, err)
	w.logger.InfoContext(
		ctx,
		"Issue generation phase completed",
		"issue_id", issueID,
		"phase", phase,
		"duration_ms", duration.Milliseconds(),
		"success", err == nil,
	)
}

func (w *Worker) renewIssueClaim(
	ctx context.Context,
	claim *store.IssueClaim,
	cancel context.CancelFunc,
	result chan<- error,
) {
	interval := min(w.cfg.ClaimDuration/3, 30*time.Second)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	currentExpiry := claim.ExpiresAt
	for {
		select {
		case <-ctx.Done():
			result <- ctx.Err()
			return
		case <-ticker.C:
			expires := w.now().Add(w.cfg.ClaimDuration)
			if err := w.lifecycle.RenewIssueClaim(
				ctx,
				claim.Issue.ID,
				claim.Token,
				expires,
			); err != nil {
				w.metrics.renewalFailures.Add(1)
				if errors.Is(err, store.ErrClaimLost) ||
					!w.now().Before(currentExpiry) {
					cancel()
					result <- err
					return
				}
				w.logger.WarnContext(
					ctx,
					"renew Issue Claim failed; retrying before expiry",
					"issue_id", claim.Issue.ID,
					"error", err,
				)
				continue
			}
			currentExpiry = expires
		}
	}
}

func (w *Worker) processDeliveries(ctx context.Context) error {
	var wg sync.WaitGroup
	errorsChannel := make(chan error, w.cfg.GlobalConcurrency)
	for count := 0; count < w.cfg.GlobalConcurrency; count++ {
		if w.draining.Load() {
			break
		}
		claim, err := w.lifecycle.ClaimNextDelivery(
			ctx,
			w.now(),
			w.cfg.ClaimDuration,
			w.cfg.MaxDeliveryAttempts,
		)
		if err != nil {
			w.captureError(err, "claim_next_delivery", nil)
			return err
		}
		if claim == nil {
			break
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.metrics.activeDeliveries.Add(1)
			defer w.metrics.activeDeliveries.Add(-1)
			if err := w.processDelivery(ctx, claim); err != nil {
				errorsChannel <- err
			}
		}()
	}
	wg.Wait()
	close(errorsChannel)
	return errors.Join(channelErrors(errorsChannel)...)
}

func (w *Worker) processDelivery(
	ctx context.Context,
	claim *store.DeliveryClaim,
) (resultErr error) {
	started := time.Now()
	defer func() {
		w.observeDuration("delivery_total", time.Since(started), resultErr)
	}()
	ctx, cancel := context.WithCancel(ctx)
	renewed := make(chan error, 1)
	go w.renewDeliveryClaim(ctx, claim, cancel, renewed)
	artifactValue, err := w.artifacts.Get(ctx, claim.Issue.ArtifactKey)
	if err == nil {
		err = w.lifecycle.RenewDeliveryClaim(
			ctx,
			claim.Issue.ID,
			claim.Token,
			w.now().Add(w.cfg.ClaimDuration),
		)
	}
	if err == nil {
		webURL := ""
		if claim.SitePublic && claim.Issue.Newsletter.SiteVisible &&
			claim.Issue.PublicationState == domain.PublicationPublished {
			webURL = fmt.Sprintf(
				"https://%s.%s/d/%s/%s",
				claim.SiteUsername,
				w.cfg.RootDomain,
				claim.Issue.PublicID,
				claim.Issue.PublicSlug,
			)
		}
		html := dossier.RenderHTML(artifactValue.Dossier, webURL)
		text := artifactValue.Markdown
		if webURL != "" {
			text += "\n\nRead on the web: " + webURL + "\n"
		}
		var externalID string
		externalID, err = w.mailer.Deliver(ctx, delivery.Message{
			IssueID: claim.Issue.ID, GenerationID: claim.Issue.GenerationID,
			To: claim.PrimaryEmail, Subject: claim.Issue.Title, HTML: html, Text: text,
		})
		if err == nil {
			completeErr := w.lifecycle.CompleteDelivery(
				ctx,
				claim.Issue.ID,
				claim.Token,
				externalID,
				w.now(),
			)
			if completeErr != nil {
				// The provider accepted the idempotent request, but the local
				// receipt was not committed. Automatic retry could duplicate
				// delivery if the provider has lost its idempotency record.
				err = &delivery.OutcomeUnknownError{Cause: completeErr}
			}
		}
	}
	cancel()
	renewErr := <-renewed
	if err == nil && renewErr != nil && !errors.Is(renewErr, context.Canceled) {
		err = renewErr
	}
	if err == nil {
		w.metrics.delivered.Add(1)
		return nil
	}
	w.metrics.deliveryFailed.Add(1)
	w.captureError(err, "deliver_issue", map[string]string{
		"issue_id": claim.Issue.ID,
		"token":    claim.Token,
	})
	var unknown *delivery.OutcomeUnknownError
	var transitionErr error
	if errors.As(err, &unknown) ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) {
		transitionErr = w.lifecycle.MarkDeliveryUnknown(
			context.Background(),
			claim.Issue.ID,
			claim.Token,
			err,
			w.now(),
		)
	} else {
		transitionErr = w.lifecycle.FailDelivery(
			context.Background(),
			claim.Issue.ID,
			claim.Token,
			err,
			w.cfg.MaxDeliveryAttempts,
			w.now(),
		)
	}
	if transitionErr != nil && !errors.Is(transitionErr, store.ErrClaimLost) {
		return errors.Join(err, transitionErr)
	}
	return fmt.Errorf("deliver Issue %s: %w", claim.Issue.ID, err)
}

func (w *Worker) observeDuration(
	operation string,
	duration time.Duration,
	err error,
) {
	outcome := "success"
	if err != nil {
		outcome = "failure"
	}
	w.metrics.durations.Observe(map[string]string{
		"operation": operation,
		"outcome":   outcome,
	}, duration.Seconds())
}

// captureError reports an operational failure to Sentry with an operation
// discriminator and any caller-provided context. It is a no-op when Sentry is
// not configured.
func (w *Worker) captureError(err error, operation string, tags map[string]string) {
	if err == nil {
		return
	}
	merged := make(map[string]string, len(tags)+1)
	for key, value := range tags {
		merged[key] = value
	}
	merged["operation"] = operation
	telemetry.CaptureError(err, merged)
}

func (w *Worker) WriteDurationMetrics(writer io.Writer) error {
	return w.metrics.durations.WritePrometheus(
		writer,
		"learnloom_worker_operation_duration_seconds",
		"Duration of worker operations in seconds.",
	)
}

func (w *Worker) renewDeliveryClaim(
	ctx context.Context,
	claim *store.DeliveryClaim,
	cancel context.CancelFunc,
	result chan<- error,
) {
	interval := min(w.cfg.ClaimDuration/3, 30*time.Second)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	currentExpiry := claim.ExpiresAt
	for {
		select {
		case <-ctx.Done():
			result <- ctx.Err()
			return
		case <-ticker.C:
			err := w.lifecycle.RenewDeliveryClaim(
				ctx,
				claim.Issue.ID,
				claim.Token,
				w.now().Add(w.cfg.ClaimDuration),
			)
			if err != nil {
				w.metrics.renewalFailures.Add(1)
				if errors.Is(err, store.ErrClaimLost) ||
					!w.now().Before(currentExpiry) {
					cancel()
					result <- err
					return
				}
				w.logger.WarnContext(
					ctx,
					"renew Delivery Claim failed; retrying before expiry",
					"issue_id", claim.Issue.ID,
					"error", err,
				)
				continue
			}
			currentExpiry = w.now().Add(w.cfg.ClaimDuration)
		}
	}
}

func (w *Worker) processDeletion(ctx context.Context) error {
	claim, err := w.lifecycle.ClaimAccountDeletion(
		ctx,
		w.now(),
		w.cfg.ClaimDuration,
	)
	if err != nil || claim == nil {
		return err
	}
	if err := w.artifacts.DeleteAccount(ctx, claim.AccountID); err != nil {
		w.captureError(err, "account_deletion", map[string]string{
			"account_id": claim.AccountID,
		})
		_ = w.lifecycle.FailAccountDeletion(
			context.Background(),
			claim.AccountID,
			claim.Token,
			err,
			w.now(),
		)
		return err
	}
	if err := w.lifecycle.CompleteAccountDeletion(
		ctx,
		claim.AccountID,
		claim.Token,
		w.now(),
	); err != nil {
		return err
	}
	w.metrics.deletions.Add(1)
	return nil
}

func (w *Worker) processArtifactCleanup(ctx context.Context) error {
	claim, err := w.lifecycle.ClaimArtifactCleanup(
		ctx, w.now(), w.cfg.ClaimDuration,
	)
	if err != nil || claim == nil {
		return err
	}
	if err := w.artifacts.Delete(ctx, claim.Key); err != nil {
		w.captureError(err, "artifact_cleanup", map[string]string{
			"artifact_key": claim.Key,
		})
		transitionErr := w.lifecycle.FailArtifactCleanup(
			context.Background(), claim.Key, claim.Token, err, w.now(),
		)
		if transitionErr != nil && !errors.Is(transitionErr, store.ErrClaimLost) {
			return errors.Join(err, transitionErr)
		}
		return fmt.Errorf("clean orphaned artifact: %w", err)
	}
	if err := w.lifecycle.CompleteArtifactCleanup(ctx, claim.Key, claim.Token); err != nil {
		return err
	}
	w.metrics.artifactCleanups.Add(1)
	return nil
}

func (w *Worker) Snapshot() Snapshot {
	last := w.metrics.lastCycleUnixNano.Load()
	var lastCycle time.Time
	if last > 0 {
		lastCycle = time.Unix(0, last).UTC()
	}
	return Snapshot{
		Cycles: w.metrics.cycles.Load(), Generated: w.metrics.generated.Load(),
		GenerationFailed:           w.metrics.generationFailed.Load(),
		GenerationDeferred:         w.metrics.generationDeferred.Load(),
		GenerationTruncated:        w.metrics.generationTruncated.Load(),
		GenerationAwaitingApproval: w.metrics.generationAwaitingApproval.Load(),
		Delivered:                  w.metrics.delivered.Load(),
		DeliveryFailed:             w.metrics.deliveryFailed.Load(),
		Deletions:                  w.metrics.deletions.Load(),
		ArtifactCleanups:           w.metrics.artifactCleanups.Load(),
		RecoveredClaims:            w.metrics.recoveredClaims.Load(),
		RenewalFailures:            w.metrics.renewalFailures.Load(),
		ReleasedClaims:             w.metrics.releasedClaims.Load(),
		ActiveIssues:               w.metrics.activeIssues.Load(),
		ActiveDeliveries:           w.metrics.activeDeliveries.Load(),
		Draining:                   w.draining.Load(),
		LastCycleAt:                lastCycle,
	}
}

func channelErrors(channel <-chan error) []error {
	var result []error
	for err := range channel {
		result = append(result, err)
	}
	return result
}
