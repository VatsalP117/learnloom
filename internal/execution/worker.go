package execution

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/VatsalP117/learnloom/internal/artifact"
	"github.com/VatsalP117/learnloom/internal/delivery"
	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/VatsalP117/learnloom/internal/dossier"
	"github.com/VatsalP117/learnloom/internal/source"
	"github.com/VatsalP117/learnloom/internal/store"
	"github.com/google/uuid"
)

type Lifecycle interface {
	RecoverExpiredClaims(context.Context, time.Time, int, int) (int64, error)
	DispatchDue(context.Context, time.Time, int) (int, error)
	ClaimNextIssue(context.Context, time.Time, time.Duration, int, int, int) (*store.IssueClaim, error)
	RenewIssueClaim(context.Context, string, string, time.Time) error
	LoadLearningHistory(context.Context, string, int) ([]domain.LearningHistoryEntry, error)
	CompleteIssue(context.Context, string, store.CompleteIssueInput) error
	FailIssue(context.Context, string, string, error, int, time.Time) error
	ClaimNextDelivery(context.Context, time.Time, time.Duration, int) (*store.DeliveryClaim, error)
	RenewDeliveryClaim(context.Context, string, string, time.Time) error
	CompleteDelivery(context.Context, string, string, string, time.Time) error
	FailDelivery(context.Context, string, string, error, int, time.Time) error
	MarkDeliveryUnknown(context.Context, string, string, error, time.Time) error
	ClaimAccountDeletion(context.Context, time.Time, time.Duration) (*store.DeletionClaim, error)
	CompleteAccountDeletion(context.Context, string, string, time.Time) error
	FailAccountDeletion(context.Context, string, string, error, time.Time) error
	CleanupOperationalState(context.Context, time.Time) (int64, error)
}

type Producer interface {
	Generate(context.Context, dossier.GenerateRequest) (dossier.GenerateResult, error)
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
	PollInterval        time.Duration
	ClaimDuration       time.Duration
	MaxIssueAttempts    int
	MaxDeliveryAttempts int
	AccountConcurrency  int
	GlobalConcurrency   int
	DailyAccountLimit   int
	DailyGlobalLimit    int
	HistoryEntries      int
	RootDomain          string
}

type Worker struct {
	lifecycle   Lifecycle
	producer    Producer
	artifacts   Artifacts
	mailer      Mailer
	sourceSvc   *source.Service
	cfg         Config
	logger      *slog.Logger
	now         func() time.Time
	lastCleanup time.Time
	metrics     workerMetrics
}

type workerMetrics struct {
	cycles            atomic.Uint64
	generated         atomic.Uint64
	generationFailed  atomic.Uint64
	delivered         atomic.Uint64
	deliveryFailed    atomic.Uint64
	deletions         atomic.Uint64
	lastCycleUnixNano atomic.Int64
}

type Snapshot struct {
	Cycles           uint64    `json:"cycles"`
	Generated        uint64    `json:"generated"`
	GenerationFailed uint64    `json:"generationFailed"`
	Delivered        uint64    `json:"delivered"`
	DeliveryFailed   uint64    `json:"deliveryFailed"`
	Deletions        uint64    `json:"deletions"`
	LastCycleAt      time.Time `json:"lastCycleAt"`
}

func New(
	lifecycle Lifecycle,
	producer Producer,
	artifacts Artifacts,
	mailer Mailer,
	sourceSvc *source.Service,
	cfg Config,
	logger *slog.Logger,
) (*Worker, error) {
	if lifecycle == nil || producer == nil || artifacts == nil || mailer == nil {
		return nil, errors.New("Issue execution dependencies are required")
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 2 * time.Second
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
	}, nil
}

func (w *Worker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()
	for {
		if err := w.Cycle(ctx); err != nil && !errors.Is(err, context.Canceled) {
			w.logger.ErrorContext(ctx, "worker cycle failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
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
		return err
	}
	if recovered > 0 {
		w.logger.WarnContext(ctx, "recovered expired claims", "count", recovered)
	}
	dispatched, err := w.lifecycle.DispatchDue(ctx, now, 100)
	if err != nil {
		return err
	}
	if dispatched > 0 {
		w.logger.InfoContext(ctx, "dispatched scheduled Issues", "count", dispatched)
	}
	if err := w.processIssues(ctx); err != nil {
		return err
	}
	if err := w.processDeliveries(ctx); err != nil {
		return err
	}
	if err := w.processDeletion(ctx); err != nil {
		return err
	}
	if w.lastCleanup.IsZero() || now.Sub(w.lastCleanup) >= time.Hour {
		if _, err := w.lifecycle.CleanupOperationalState(
			ctx,
			now.Add(-30*24*time.Hour),
		); err != nil {
			return err
		}
		w.lastCleanup = now
	}
	return nil
}

func (w *Worker) processIssues(ctx context.Context) error {
	var wg sync.WaitGroup
	errorsChannel := make(chan error, w.cfg.GlobalConcurrency)
	for count := 0; count < w.cfg.GlobalConcurrency; count++ {
		claim, err := w.lifecycle.ClaimNextIssue(
			ctx,
			w.now(),
			w.cfg.ClaimDuration,
			w.cfg.AccountConcurrency,
			w.cfg.DailyAccountLimit,
			w.cfg.DailyGlobalLimit,
		)
		if errors.Is(err, store.ErrGenerationPaused) ||
			errors.Is(err, store.ErrQuotaExceeded) {
			break
		}
		if err != nil {
			return err
		}
		if claim == nil {
			break
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
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
	ctx, cancel := context.WithCancel(ctx)
	renewed := make(chan error, 1)
	go w.renewIssueClaim(ctx, claim, cancel, renewed)
	var err error
	history, err := w.lifecycle.LoadLearningHistory(
		ctx,
		claim.Issue.NewsletterID,
		w.cfg.HistoryEntries,
	)
	if err == nil {
		if w.sourceSvc == nil {
			err = errors.New("source intelligence service is unavailable")
		}
	}
	if err == nil {
		var prepared source.PrepareIssueResult
		prepared, err = w.sourceSvc.PrepareIssue(ctx, claim.Issue.Newsletter, claim.Issue.ID)
		if err == nil {
			var result dossier.GenerateResult
			result, err = w.producer.Generate(ctx, dossier.GenerateRequest{
				Newsletter: claim.Issue.Newsletter,
				History:    history,
				Now:        w.now(),
				OnStage: func(stage string) {
					w.logger.InfoContext(
						ctx,
						"Dossier stage",
						"issue_id", claim.Issue.ID,
						"stage", stage,
					)
				},
				PreparedItems: prepared.Items,
			})
			if err == nil {
				generationID := uuid.NewString()
				var saved artifact.PutResult
				saved, err = w.artifacts.Put(ctx, artifact.PutInput{
					AccountID: claim.AccountID, NewsletterID: claim.Issue.NewsletterID,
					IssueID: claim.Issue.ID, GenerationID: generationID,
					Artifact: result.Artifact,
				})
				if err == nil {
					err = w.lifecycle.CompleteIssue(ctx, claim.Issue.ID, store.CompleteIssueInput{
						ClaimToken: claim.Token, GenerationID: generationID,
						ArtifactKey: saved.Key, Checksum: saved.Checksum, Bytes: saved.Bytes,
						Title: result.Artifact.Dossier.Title, History: result.History,
						HistoryLimit: w.cfg.HistoryEntries, CompletedAt: w.now(),
					})
					if err != nil {
						_ = w.artifacts.Delete(context.Background(), saved.Key)
					}
				}
			}
		}
	}
	cancel()
	renewErr := <-renewed
	if err == nil && renewErr != nil && !errors.Is(renewErr, context.Canceled) {
		err = renewErr
	}
	if err != nil {
		w.metrics.generationFailed.Add(1)
		failErr := w.lifecycle.FailIssue(
			context.Background(),
			claim.Issue.ID,
			claim.Token,
			err,
			w.cfg.MaxIssueAttempts,
			w.now(),
		)
		if failErr != nil && !errors.Is(failErr, store.ErrClaimLost) {
			return errors.Join(err, failErr)
		}
		return fmt.Errorf("generate Issue %s: %w", claim.Issue.ID, err)
	}
	w.metrics.generated.Add(1)
	return nil
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
				cancel()
				result <- err
				return
			}
		}
	}
}

func (w *Worker) processDeliveries(ctx context.Context) error {
	var wg sync.WaitGroup
	errorsChannel := make(chan error, w.cfg.GlobalConcurrency)
	for count := 0; count < w.cfg.GlobalConcurrency; count++ {
		claim, err := w.lifecycle.ClaimNextDelivery(
			ctx,
			w.now(),
			w.cfg.ClaimDuration,
			w.cfg.MaxDeliveryAttempts,
		)
		if err != nil {
			return err
		}
		if claim == nil {
			break
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
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
) error {
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
	var unknown *delivery.OutcomeUnknownError
	var transitionErr error
	if errors.As(err, &unknown) {
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

func (w *Worker) renewDeliveryClaim(
	ctx context.Context,
	claim *store.DeliveryClaim,
	cancel context.CancelFunc,
	result chan<- error,
) {
	interval := min(w.cfg.ClaimDuration/3, 30*time.Second)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
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
				cancel()
				result <- err
				return
			}
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

func (w *Worker) Snapshot() Snapshot {
	last := w.metrics.lastCycleUnixNano.Load()
	var lastCycle time.Time
	if last > 0 {
		lastCycle = time.Unix(0, last).UTC()
	}
	return Snapshot{
		Cycles: w.metrics.cycles.Load(), Generated: w.metrics.generated.Load(),
		GenerationFailed: w.metrics.generationFailed.Load(),
		Delivered:        w.metrics.delivered.Load(),
		DeliveryFailed:   w.metrics.deliveryFailed.Load(),
		Deletions:        w.metrics.deletions.Load(), LastCycleAt: lastCycle,
	}
}

func channelErrors(channel <-chan error) []error {
	var result []error
	for err := range channel {
		result = append(result, err)
	}
	return result
}
