package execution

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/VatsalP117/learnloom/internal/artifact"
	"github.com/VatsalP117/learnloom/internal/delivery"
	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/VatsalP117/learnloom/internal/dossier"
	"github.com/VatsalP117/learnloom/internal/store"
)

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

func validConfig() Config {
	return Config{
		PollInterval: time.Second, ClaimDuration: time.Minute,
		MaxIssueAttempts: 3, MaxDeliveryAttempts: 3,
		AccountConcurrency: 1, GlobalConcurrency: 1,
		DailyAccountLimit: 5, DailyGlobalLimit: 100,
		HistoryEntries: 10, RootDomain: "learnloom.blog",
	}
}
