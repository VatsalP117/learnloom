package store

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/google/uuid"
)

func TestIssueHistoryQueryPlanIntegration(t *testing.T) {
	database := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	account, err := database.SyncAccountIdentity(
		ctx,
		"clerk-plan-"+uuid.NewString(),
		"plan@example.com",
		domain.AccountActive,
		time.Now().UTC().UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	requirePaidIntegrationPlan(t, ctx, database, account.ID, time.Now().UTC())
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
			Name: "Plan source", URL: "https://example.com/feed.xml", Limit: 5,
		}}),
		10,
		5,
	)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := database.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer rollback(tx)
	if _, err := tx.Exec(ctx, "SET LOCAL enable_seqscan = off"); err != nil {
		t.Fatal(err)
	}
	var raw []byte
	if err := tx.QueryRow(ctx, `
		EXPLAIN (ANALYZE, FORMAT JSON)
		SELECT id
		FROM issues
		WHERE newsletter_id = $1
		ORDER BY created_at DESC
		LIMIT 40
	`, created.Newsletter.ID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var plan []struct {
		ExecutionTime float64 `json:"Execution Time"`
	}
	if err := json.Unmarshal(raw, &plan); err != nil || len(plan) != 1 {
		t.Fatalf("decode query plan: %s err=%v", raw, err)
	}
	if !strings.Contains(string(raw), "issues_newsletter_history") {
		t.Fatalf("history query lost its intended index: %s", raw)
	}
	if plan[0].ExecutionTime > 250 {
		t.Fatalf("history query exceeded 250ms budget: %.3fms", plan[0].ExecutionTime)
	}
}
