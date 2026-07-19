package store

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/google/uuid"
)

func TestPostgresLifecycleIntegration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	database, err := Open(ctx, Config{URL: databaseURL, MaxConnections: 5})
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := database.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := database.Ready(ctx); err != nil {
		t.Fatal(err)
	}
	identityTime := time.Now().UTC().UnixMilli()
	account, err := database.SyncAccountIdentity(
		ctx,
		"clerk-test-"+uuid.NewString(),
		"learner@example.com",
		domain.AccountActive,
		identityTime,
	)
	if err != nil {
		t.Fatal(err)
	}
	site, err := database.ClaimSite(
		ctx,
		account.ID,
		"learner-"+uuid.NewString()[:8],
		"Learner",
	)
	if err != nil {
		t.Fatal(err)
	}
	site, err = database.UpdateSite(
		ctx,
		account.ID,
		domain.SitePublic,
		nil,
		nil,
	)
	if err != nil || site.Visibility != domain.SitePublic {
		t.Fatalf("site=%#v err=%v", site, err)
	}
	newsletter, err := database.CreateNewsletter(
		ctx,
		account.ID,
		NewsletterInput{
			Name: "Systems", Topic: "software systems", LearnerLevel: "experienced",
			LearnerGoal: "build durable understanding", LessonMinutes: 15,
			Sources: []domain.SourceDefinition{{
				Name: "Example", URL: "https://example.com/feed.xml", Limit: 5,
			}},
			ScheduleHour: 9, TimeZone: "Asia/Kolkata", Active: true,
			EmailEnabled: true, SiteVisible: true,
		},
		10,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.SetNewsletterActive(ctx, account.ID, newsletter.ID, false); err != nil {
		t.Fatal(err)
	}
	issue, err := database.EnqueueManualIssue(ctx, account.ID, newsletter.ID, 5)
	if err != nil {
		t.Fatal(err)
	}
	claim, err := database.ClaimNextIssue(
		ctx,
		time.Now().UTC(),
		5*time.Minute,
		1,
		5,
		100,
	)
	if err != nil || claim == nil || claim.Issue.ID != issue.ID {
		t.Fatalf("claim=%#v err=%v", claim, err)
	}
	now := time.Now().UTC()
	err = database.CompleteIssue(ctx, issue.ID, CompleteIssueInput{
		ClaimToken: claim.Token, GenerationID: uuid.NewString(),
		ArtifactKey: "accounts/a/dossier.json", Checksum: "abc", Bytes: 100,
		Title: "A generated Dossier",
		History: domain.LearningHistoryEntry{
			Date: "2026-07-19", GeneratedAt: now,
			LessonSummary: "Summary", LearningObjective: "Objective",
		},
		HistoryLimit: 14, CompletedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	history, err := database.LoadLearningHistory(ctx, newsletter.ID, 14)
	if err != nil || len(history) != 1 {
		t.Fatalf("history=%#v err=%v", history, err)
	}
	publicIssues, err := database.ListPublicIssues(ctx, site.Username, "", 10)
	if err != nil || len(publicIssues) != 1 {
		t.Fatalf("publicIssues=%#v err=%v", publicIssues, err)
	}
	deliveryClaim, err := database.ClaimNextDelivery(
		ctx,
		time.Now().UTC(),
		5*time.Minute,
		6,
	)
	if err != nil || deliveryClaim == nil || deliveryClaim.PrimaryEmail != "learner@example.com" {
		t.Fatalf("deliveryClaim=%#v err=%v", deliveryClaim, err)
	}
	if err := database.CompleteDelivery(
		ctx,
		issue.ID,
		deliveryClaim.Token,
		"resend-test",
		time.Now().UTC(),
	); err != nil {
		t.Fatal(err)
	}

	other, err := database.SyncAccountIdentity(
		ctx,
		"clerk-other-"+uuid.NewString(),
		"other@example.com",
		domain.AccountActive,
		identityTime,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.GetNewsletter(ctx, other.ID, newsletter.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-Account Newsletter read was not denied: %v", err)
	}
	if _, err := database.EnqueueManualIssue(ctx, other.ID, newsletter.ID, 5); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-Account Issue creation was not denied: %v", err)
	}

	window := time.Now().UTC()
	allowed, err := database.AllowRequest(ctx, account.ID, "integration", time.Minute, 1, window)
	if err != nil || !allowed {
		t.Fatalf("initial request limit failed: allowed=%v err=%v", allowed, err)
	}
	allowed, err = database.AllowRequest(ctx, account.ID, "integration", time.Minute, 1, window)
	if err != nil || allowed {
		t.Fatalf("request limit did not reject excess work: allowed=%v err=%v", allowed, err)
	}

	eventID := "event-" + uuid.NewString()
	fresh, err := database.BeginWebhook(ctx, eventID, "user.updated", window)
	if err != nil || !fresh {
		t.Fatalf("begin webhook: fresh=%v err=%v", fresh, err)
	}
	fresh, err = database.BeginWebhook(ctx, eventID, "user.updated", window)
	if err != nil || fresh {
		t.Fatalf("concurrent duplicate webhook was not suppressed: fresh=%v err=%v", fresh, err)
	}
	if err := database.CompleteWebhook(ctx, eventID, errors.New("retry"), window); err != nil {
		t.Fatal(err)
	}
	fresh, err = database.BeginWebhook(ctx, eventID, "user.updated", window)
	if err != nil || !fresh {
		t.Fatalf("failed webhook was not retryable: fresh=%v err=%v", fresh, err)
	}
	if err := database.CompleteWebhook(ctx, eventID, nil, window); err != nil {
		t.Fatal(err)
	}
	abandonedID := "event-" + uuid.NewString()
	fresh, err = database.BeginWebhook(ctx, abandonedID, "user.updated", window)
	if err != nil || !fresh {
		t.Fatalf("begin abandoned webhook: fresh=%v err=%v", fresh, err)
	}
	fresh, err = database.BeginWebhook(
		ctx,
		abandonedID,
		"user.updated",
		window.Add(6*time.Minute),
	)
	if err != nil || !fresh {
		t.Fatalf("abandoned webhook Claim did not expire: fresh=%v err=%v", fresh, err)
	}

	if _, err := database.SyncAccountIdentity(
		ctx,
		account.ClerkUserID,
		"",
		domain.AccountDeleted,
		identityTime+10,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := database.EnsureAccount(ctx, account.ClerkUserID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("deleted Account retained access: %v", err)
	}
	staleUpdate, err := database.SyncAccountIdentity(
		ctx,
		account.ClerkUserID,
		"learner@example.com",
		domain.AccountActive,
		identityTime+5,
	)
	if err != nil || staleUpdate.Status != domain.AccountDeleted {
		t.Fatalf("stale webhook revived deleted Account: %#v %v", staleUpdate, err)
	}
	publicIssues, err = database.ListPublicIssues(ctx, site.Username, "", 10)
	if err != nil || len(publicIssues) != 0 {
		t.Fatalf("deleted Account retained public content: %#v %v", publicIssues, err)
	}
	deletion, err := database.ClaimAccountDeletion(ctx, time.Now().UTC(), 5*time.Minute)
	if err != nil || deletion == nil || deletion.AccountID != account.ID {
		t.Fatalf("artifact deletion was not queued: %#v %v", deletion, err)
	}
}
