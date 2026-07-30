package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/google/uuid"
)

func TestModelBudgetClaimEnforcementIntegration(t *testing.T) {
	database := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	account, err := database.SyncAccountIdentity(
		ctx,
		"clerk-budget-"+uuid.NewString(),
		"budget@example.com",
		domain.AccountActive,
		time.Now().UTC().UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = database.pool.Exec(
			context.Background(),
			"DELETE FROM accounts WHERE id = $1",
			account.ID,
		)
	})
	created, err := database.CreateNewsletter(
		ctx,
		account.ID,
		integrationNewsletterInput([]domain.SourceDefinition{{
			Name: "Budget source", URL: "https://example.com/feed.xml", Limit: 5,
		}}),
		10,
		5,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.pool.Exec(ctx, `
		UPDATE issues
		SET available_at = '2000-01-01T00:00:00Z',
		    created_at = '2000-01-01T00:00:00Z'
		WHERE id = $1
	`, created.FirstIssue.ID); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2100, 1, 2, 10, 0, 0, 0, time.UTC)
	claim, err := database.ClaimNextIssue(
		ctx, now, 5*time.Minute, 1, 5, 100,
		100, 100,
		IssueAttemptContext{WorkerID: "budget-test"},
	)
	if err != nil || claim == nil || claim.Issue.ID != created.FirstIssue.ID {
		t.Fatalf("initial budgeted claim=%#v err=%v", claim, err)
	}
	if err := database.ReleaseIssueClaim(
		ctx,
		claim.Issue.ID,
		claim.Token,
		errors.New("release for budget test"),
		now.Add(time.Second),
	); err != nil {
		t.Fatal(err)
	}
	if next, err := database.ClaimNextIssue(
		ctx, now.Add(2*time.Second), 5*time.Minute, 1, 5, 100,
		100, 100,
		IssueAttemptContext{WorkerID: "budget-test"},
	); !errors.Is(err, ErrQuotaExceeded) || next != nil {
		t.Fatalf("budget cap next=%#v err=%v", next, err)
	}
}
