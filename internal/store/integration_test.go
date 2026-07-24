package store

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
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
			SourceMode: domain.SourceModeProvided,
			Sources: []domain.SourceDefinition{{
				Name: "Example", URL: "https://example.com/feed.xml", Limit: 5,
			}},
			ScheduleHour: 9, TimeZone: "Asia/Kolkata", Active: true,
			EmailEnabled: true, SiteVisible: true,
		},
		10,
		5,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.SetNewsletterActive(ctx, account.ID, newsletter.Newsletter.ID, false); err != nil {
		t.Fatal(err)
	}
	issue := newsletter.FirstIssue
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
	if err := database.FailIssue(
		ctx,
		issue.ID,
		claim.Token,
		errors.New("editor output contract failed"),
		3,
		now,
	); err != nil {
		t.Fatalf("fail Issue: %v", err)
	}
	claim, err = database.ClaimNextIssue(
		ctx,
		now.Add(16*time.Second),
		5*time.Minute,
		1,
		5,
		100,
	)
	if err != nil || claim == nil || claim.Issue.ID != issue.ID {
		t.Fatalf("retry claim=%#v err=%v", claim, err)
	}
	if err := database.FailIssue(
		ctx,
		issue.ID,
		claim.Token,
		errors.New("editor output contract failed again"),
		1,
		now.Add(17*time.Second),
	); err != nil {
		t.Fatalf("exhaust Issue attempts: %v", err)
	}
	if err := database.RetryIssue(
		ctx,
		account.ID,
		issue.ID,
		now.Add(18*time.Second),
	); err != nil {
		t.Fatalf("retry failed Issue: %v", err)
	}
	claim, err = database.ClaimNextIssue(
		ctx,
		now.Add(19*time.Second),
		5*time.Minute,
		1,
		5,
		100,
	)
	if err != nil || claim == nil || claim.Issue.ID != issue.ID {
		t.Fatalf("manual retry claim=%#v err=%v", claim, err)
	}
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
	history, err := database.LoadLearningHistory(ctx, newsletter.Newsletter.ID, 14)
	if err != nil || len(history) != 1 {
		t.Fatalf("history=%#v err=%v", history, err)
	}
	reviews, err := database.ListWorkspaceReviews(ctx, account.ID, 8)
	if err != nil || len(reviews) != 1 || reviews[0].IssueID != issue.ID {
		t.Fatalf("workspace reviews=%#v err=%v", reviews, err)
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
	newerIssue, err := database.EnqueueManualIssue(
		ctx,
		account.ID,
		newsletter.Newsletter.ID,
		5,
	)
	if err != nil {
		t.Fatal(err)
	}
	page, cursor, err := database.ListWorkspaceIssuesPage(ctx, account.ID, 1, nil)
	if err != nil || len(page) != 1 || page[0].ID != newerIssue.ID || cursor == nil {
		t.Fatalf("first workspace page=%#v cursor=%#v err=%v", page, cursor, err)
	}
	older, finalCursor, err := database.ListWorkspaceIssuesPage(ctx, account.ID, 1, cursor)
	if err != nil || len(older) != 1 || older[0].ID != issue.ID || finalCursor != nil {
		t.Fatalf("second workspace page=%#v cursor=%#v err=%v", older, finalCursor, err)
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
	if _, err := database.GetNewsletter(ctx, other.ID, newsletter.Newsletter.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-Account Newsletter read was not denied: %v", err)
	}
	if _, err := database.EnqueueManualIssue(ctx, other.ID, newsletter.Newsletter.ID, 5); !errors.Is(err, ErrNotFound) {
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

func TestSourceCatalogReconciliationIntegration(t *testing.T) {
	database := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	account, err := database.SyncAccountIdentity(
		ctx,
		"clerk-catalog-"+uuid.NewString(),
		"catalog@example.com",
		domain.AccountActive,
		time.Now().UTC().UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	input := integrationNewsletterInput([]domain.SourceDefinition{
		{Name: "First", URL: "https://example.com/first", Limit: 5},
		{Name: "Remove me", URL: "https://example.com/remove", Limit: 6},
	})
	created, err := database.CreateNewsletter(ctx, account.ID, input, 10, 5)
	if err != nil {
		t.Fatal(err)
	}
	discovered, err := database.UpsertDiscoveredSourceSpec(ctx, domain.SourceSpec{
		ID:              uuid.NewString(),
		NewsletterID:    created.Newsletter.ID,
		Origin:          domain.SourceOriginDiscovered,
		State:           domain.SourceStateActive,
		DisplayName:     "Discovered reference",
		InputURL:        "https://docs.example.org/guide",
		CanonicalURL:    "https://docs.example.org/guide",
		Scope:           domain.SourceScopeExact,
		ItemLimit:       8,
		DiscoveryReason: "official reference",
	})
	if err != nil {
		t.Fatal(err)
	}
	input.Sources = []domain.SourceDefinition{
		{Name: "First renamed", URL: "https://example.com/first", Limit: 9},
		{Name: "Added", URL: "https://example.com/added", Limit: 4},
	}
	updated, err := database.UpdateNewsletter(ctx, account.ID, created.Newsletter.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(updated.Sources, input.Sources) {
		t.Fatalf("compatibility sources=%#v, want %#v", updated.Sources, input.Sources)
	}
	rows, err := database.pool.Query(ctx, `
		SELECT canonical_url, display_name, item_limit, state
		FROM source_specs
		WHERE newsletter_id = $1 AND origin = 'provided'
		ORDER BY canonical_url
	`, created.Newsletter.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	type catalogRow struct {
		url, name, state string
		limit            int
	}
	var got []catalogRow
	for rows.Next() {
		var row catalogRow
		if err := rows.Scan(&row.url, &row.name, &row.limit, &row.state); err != nil {
			t.Fatal(err)
		}
		got = append(got, row)
	}
	want := []catalogRow{
		{url: "https://example.com/added", name: "Added", limit: 4, state: "active"},
		{url: "https://example.com/first", name: "First renamed", limit: 9, state: "active"},
		{url: "https://example.com/remove", name: "Remove me", limit: 6, state: "disabled"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("catalog=%#v, want %#v", got, want)
	}
	var discoveredState domain.SourceState
	if err := database.pool.QueryRow(ctx, `
		SELECT state FROM source_specs WHERE id = $1
	`, discovered.ID).Scan(&discoveredState); err != nil {
		t.Fatal(err)
	}
	if discoveredState != domain.SourceStateDisabled {
		t.Fatalf("discovered state after provided update=%q", discoveredState)
	}
	input.SourceMode = domain.SourceModeHybrid
	if _, err := database.UpdateNewsletter(
		ctx,
		account.ID,
		created.Newsletter.ID,
		input,
	); err != nil {
		t.Fatal(err)
	}
	if err := database.pool.QueryRow(ctx, `
		SELECT state FROM source_specs WHERE id = $1
	`, discovered.ID).Scan(&discoveredState); err != nil {
		t.Fatal(err)
	}
	if discoveredState != domain.SourceStateActive {
		t.Fatalf("discovered state after hybrid update=%q", discoveredState)
	}
}

func TestCreateNewsletterDailyQuotaRollsBackIntegration(t *testing.T) {
	database := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	account, err := database.SyncAccountIdentity(
		ctx,
		"clerk-quota-"+uuid.NewString(),
		"quota@example.com",
		domain.AccountActive,
		time.Now().UTC().UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	input := integrationNewsletterInput([]domain.SourceDefinition{{
		Name: "One", URL: "https://example.com/one", Limit: 5,
	}})
	if _, err := database.CreateNewsletter(ctx, account.ID, input, 10, 1); err != nil {
		t.Fatal(err)
	}
	input.Name = "Should roll back"
	input.Topic = "a different topic"
	input.Sources = []domain.SourceDefinition{{
		Name: "Two", URL: "https://example.com/two", Limit: 5,
	}}
	if _, err := database.CreateNewsletter(ctx, account.ID, input, 10, 1); !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("err=%v, want ErrQuotaExceeded", err)
	}
	var newsletters, specs, issues int
	if err := database.pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM newsletters WHERE owner_account_id = $1),
			(SELECT count(*) FROM source_specs ss JOIN newsletters n ON n.id = ss.newsletter_id WHERE n.owner_account_id = $1),
			(SELECT count(*) FROM issues i JOIN newsletters n ON n.id = i.newsletter_id WHERE n.owner_account_id = $1)
	`, account.ID).Scan(&newsletters, &specs, &issues); err != nil {
		t.Fatal(err)
	}
	if newsletters != 1 || specs != 1 || issues != 1 {
		t.Fatalf("rollback counts newsletters=%d specs=%d issues=%d", newsletters, specs, issues)
	}
}

func TestSourceEvidenceAndDiscoveryRepositoryIntegration(t *testing.T) {
	database := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	account, err := database.SyncAccountIdentity(
		ctx,
		"clerk-source-repo-"+uuid.NewString(),
		"source-repo@example.com",
		domain.AccountActive,
		time.Now().UTC().UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	input := integrationNewsletterInput(nil)
	input.SourceMode = domain.SourceModeDiscovered
	created, err := database.CreateNewsletter(ctx, account.ID, input, 10, 5)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := database.UpsertDiscoveredSourceSpec(ctx, domain.SourceSpec{
		ID: uuid.NewString(), NewsletterID: created.Newsletter.ID,
		Origin: domain.SourceOriginDiscovered, State: domain.SourceStateCandidate,
		DisplayName: "Discovered guide", InputURL: "https://example.com/guide",
		CanonicalURL: "https://example.com/guide", Scope: domain.SourceScopeExact,
		ItemLimit: 8, DiscoveryReason: "official reference",
		DiscoveryQuery: "topic official documentation", RankScore: 42,
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	endpoint, err := database.UpsertSourceEndpoint(ctx, domain.SourceEndpoint{
		ID: uuid.NewString(), SourceSpecID: spec.ID,
		EndpointURL: spec.InputURL, CanonicalURL: spec.CanonicalURL,
		Kind: domain.SourceKindHTML, Health: "healthy",
		LastCheckedAt: &now, LastSuccessAt: &now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := domain.SourceSnapshot{
		ID: uuid.NewString(), SourceEndpointID: endpoint.ID,
		ItemKey: spec.CanonicalURL, Title: spec.DisplayName,
		CanonicalURL: spec.CanonicalURL, Content: strings.Repeat("evidence ", 100),
		ContentSource: "article", ContentSHA256: "content-hash",
		Metadata:  `{"source":"Discovered guide","origin":"discovered"}`,
		FetchedAt: now,
	}
	firstID, err := database.InsertSourceSnapshot(ctx, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.ID = uuid.NewString()
	secondID, err := database.InsertSourceSnapshot(ctx, snapshot)
	if err != nil || secondID != firstID {
		t.Fatalf("snapshot idempotency first=%q second=%q err=%v", firstID, secondID, err)
	}
	inserted, err := database.InsertIssueSources(ctx, created.FirstIssue.ID, []domain.IssueSource{{
		IssueID: created.FirstIssue.ID, SourceSnapshotID: firstID,
		Position: 0, CreatedAt: now,
	}})
	if err != nil || !inserted {
		t.Fatalf("freeze inserted=%v err=%v", inserted, err)
	}
	inserted, err = database.InsertIssueSources(ctx, created.FirstIssue.ID, []domain.IssueSource{{
		IssueID: created.FirstIssue.ID, SourceSnapshotID: firstID,
		Position: 0, CreatedAt: now,
	}})
	if err != nil || inserted {
		t.Fatalf("second freeze inserted=%v err=%v", inserted, err)
	}
	frozen, err := database.GetIssueSources(ctx, created.FirstIssue.ID)
	if err != nil || len(frozen) != 1 || frozen[0].ID != firstID {
		t.Fatalf("frozen=%#v err=%v", frozen, err)
	}
	if err := database.SetSourceSpecState(
		ctx,
		spec.ID,
		domain.SourceStateActive,
		domain.SourceKindHTML,
	); err != nil {
		t.Fatal(err)
	}
	started := now
	run := domain.DiscoveryRun{
		ID: uuid.NewString(), NewsletterID: created.Newsletter.ID,
		IssueID: created.FirstIssue.ID, Reason: "initial", State: "running",
		QueryBundle: `["topic official documentation"]`, StartedAt: &started,
	}
	if err := database.CreateDiscoveryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	completed := now.Add(time.Second)
	run.State = "completed"
	run.ReturnedCandidates = 3
	run.ResolvedCandidates = 1
	run.ActivatedCandidates = 1
	run.CompletedAt = &completed
	if err := database.CompleteDiscoveryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	catalog, err := database.ListSourceCatalog(ctx, account.ID, created.Newsletter.ID, 50)
	if err != nil || len(catalog) != 1 || catalog[0].Origin != domain.SourceOriginDiscovered ||
		catalog[0].Health != "healthy" {
		t.Fatalf("catalog=%#v err=%v", catalog, err)
	}
	other, err := database.SyncAccountIdentity(
		ctx,
		"clerk-source-other-"+uuid.NewString(),
		"source-other@example.com",
		domain.AccountActive,
		time.Now().UTC().UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if catalog, err := database.ListSourceCatalog(ctx, other.ID, created.Newsletter.ID, 50); err != nil || len(catalog) != 0 {
		t.Fatalf("cross-account catalog=%#v err=%v", catalog, err)
	}
}

func openIntegrationStore(t *testing.T) *Store {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	database, err := Open(ctx, Config{URL: databaseURL, MaxConnections: 5})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(database.Close)
	if err := database.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	return database
}

func integrationNewsletterInput(sources []domain.SourceDefinition) NewsletterInput {
	return NewsletterInput{
		Name: "Source integration", Topic: "source intelligence",
		LearnerLevel: "intermediate", LearnerGoal: "build a practical understanding",
		LessonMinutes: 20, SourceMode: domain.SourceModeProvided, Sources: sources,
		ScheduleHour: 8, TimeZone: "UTC", Active: true,
	}
}
