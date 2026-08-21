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
	"github.com/VatsalP117/learnloom/internal/failure"
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
	var transitionalSourcesColumn bool
	if err := database.pool.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1
		  FROM information_schema.columns
		  WHERE table_schema = 'public'
		    AND table_name = 'newsletters'
		    AND column_name = 'sources'
		)
	`).Scan(&transitionalSourcesColumn); err != nil {
		t.Fatal(err)
	}
	if transitionalSourcesColumn {
		t.Fatal("transitional newsletters.sources column still exists")
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
	requirePaidIntegrationPlan(t, ctx, database, account.ID, time.Now().UTC())
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
		nil,
	)
	if err != nil || site.Visibility != domain.SitePublic || site.SearchIndexing {
		t.Fatalf("site=%#v err=%v", site, err)
	}
	searchIndexing := true
	site, err = database.UpdateSite(
		ctx,
		account.ID,
		domain.SitePublic,
		nil,
		nil,
		&searchIndexing,
	)
	if err != nil || !site.SearchIndexing {
		t.Fatalf("site search indexing=%#v err=%v", site, err)
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
			TemplateID: "ai-systems-evidence", TemplateVersion: 1,
		},
		10,
		5,
	)
	if err != nil {
		t.Fatal(err)
	}
	var templateID string
	var templateVersion int
	if err := database.pool.QueryRow(ctx, `
		SELECT stream_template_id, stream_template_version
		FROM newsletters
		WHERE id = $1 AND owner_account_id = $2
	`, newsletter.Newsletter.ID, account.ID).Scan(
		&templateID,
		&templateVersion,
	); err != nil || templateID != "ai-systems-evidence" || templateVersion != 1 {
		t.Fatalf(
			"template attribution=%q v%d err=%v",
			templateID,
			templateVersion,
			err,
		)
	}
	if err := database.SetNewsletterActive(ctx, account.ID, newsletter.Newsletter.ID, false); err != nil {
		t.Fatal(err)
	}
	issue := newsletter.FirstIssue
	duplicate, err := database.EnqueueManualIssue(
		ctx,
		account.ID,
		newsletter.Newsletter.ID,
		5,
	)
	if err != nil || duplicate.ID != issue.ID {
		t.Fatalf("active manual Issue was not idempotent: issue=%#v err=%v", duplicate, err)
	}
	claim, err := database.ClaimNextIssue(
		ctx,
		time.Now().UTC(),
		5*time.Minute,
		1,
		5,
		100,
		10_000_000,
		1_000_000,
		IssueAttemptContext{WorkerID: "test-worker", DeploymentVersion: "test"},
	)
	if err != nil || claim == nil || claim.Issue.ID != issue.ID {
		t.Fatalf("claim=%#v err=%v", claim, err)
	}
	if err := database.RecordIssueStage(
		ctx,
		issue.ID,
		claim.Token,
		"curator",
		150*time.Millisecond,
		StageUsage{
			InputTokens: 100, OutputTokens: 50,
			ProviderRetries: 1, EstimatedCostMicroUSD: 200,
		},
		nil,
		time.Now().UTC(),
	); err != nil {
		t.Fatalf("record model economics: %v", err)
	}
	if err := database.SaveIssueCheckpoint(
		ctx,
		issue.ID,
		claim.Token,
		"fingerprint-1",
		"blueprint",
		`{"learningObjective":"test"}`,
		"dossier-v3",
		time.Now().UTC(),
	); err != nil {
		t.Fatalf("save Issue checkpoint: %v", err)
	}
	checkpoints, err := database.LoadIssueCheckpoints(ctx, issue.ID, "fingerprint-1")
	if err != nil || checkpoints["blueprint"] == "" {
		t.Fatalf("load Issue checkpoints: %#v err=%v", checkpoints, err)
	}
	expiredAt := claim.ExpiresAt.Add(time.Second)
	recovered, err := database.RecoverExpiredClaims(ctx, expiredAt, 3, 6)
	if err != nil || recovered != 1 {
		t.Fatalf("recover expired claim: recovered=%d err=%v", recovered, err)
	}
	var issueStatus string
	var issueAttempts, claimLosses int
	var attemptStatus, attemptIncident, issueIncident, attemptWorker, attemptDeployment string
	if err := database.pool.QueryRow(ctx, `
		SELECT status, attempt_count, claim_loss_count, incident_id::text
		FROM issues WHERE id = $1
	`, issue.ID).Scan(
		&issueStatus,
		&issueAttempts,
		&claimLosses,
		&issueIncident,
	); err != nil {
		t.Fatal(err)
	}
	if err := database.pool.QueryRow(ctx, `
		SELECT status, incident_id::text, worker_id, deployment_version
		FROM issue_attempts WHERE id = $1
	`, claim.Token).Scan(
		&attemptStatus,
		&attemptIncident,
		&attemptWorker,
		&attemptDeployment,
	); err != nil {
		t.Fatal(err)
	}
	if issueStatus != "queued" || issueAttempts != 0 || claimLosses != 1 ||
		attemptStatus != "abandoned" || attemptIncident != issueIncident {
		t.Fatalf(
			"unexpected recovery: issue=%s attempts=%d losses=%d attempt=%s incidents=%s/%s",
			issueStatus, issueAttempts, claimLosses, attemptStatus,
			attemptIncident, issueIncident,
		)
	}
	if attemptWorker != "test-worker" || attemptDeployment != "test" {
		t.Fatalf("attempt identity=%s/%s", attemptWorker, attemptDeployment)
	}
	claim, err = database.ClaimNextIssue(
		ctx,
		expiredAt.Add(16*time.Second),
		5*time.Minute,
		1,
		5,
		100,
		10_000_000,
		1_000_000,
		IssueAttemptContext{WorkerID: "test-worker", DeploymentVersion: "test"},
	)
	if err != nil || claim == nil || claim.Issue.ID != issue.ID {
		t.Fatalf("recovery claim=%#v err=%v", claim, err)
	}
	now := expiredAt.Add(17 * time.Second)
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
		10_000_000,
		1_000_000,
		IssueAttemptContext{WorkerID: "test-worker", DeploymentVersion: "test"},
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
	failedIssue, err := database.GetIssue(ctx, account.ID, issue.ID)
	if err != nil {
		t.Fatal(err)
	}
	if failedIssue.Status != domain.IssueFailed ||
		failedIssue.FailureCode != "internal_error" ||
		failedIssue.Error == "" ||
		strings.Contains(failedIssue.Error, "editor output contract failed") {
		t.Fatalf("unsafe failed Issue projection: %#v", failedIssue)
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
		10_000_000,
		1_000_000,
		IssueAttemptContext{WorkerID: "test-worker", DeploymentVersion: "test"},
	)
	if err != nil || claim == nil || claim.Issue.ID != issue.ID {
		t.Fatalf("manual retry claim=%#v err=%v", claim, err)
	}
	if err := database.ReleaseIssueClaim(
		ctx,
		issue.ID,
		claim.Token,
		errors.New("worker draining"),
		now.Add(20*time.Second),
	); err != nil {
		t.Fatalf("release Issue Claim: %v", err)
	}
	claim, err = database.ClaimNextIssue(
		ctx,
		now.Add(36*time.Second),
		5*time.Minute,
		1,
		5,
		100,
		10_000_000,
		1_000_000,
		IssueAttemptContext{WorkerID: "test-worker", DeploymentVersion: "test"},
	)
	if err != nil || claim == nil || claim.Issue.ID != issue.ID {
		t.Fatalf("released retry claim=%#v err=%v", claim, err)
	}
	committedArtifactKey := "accounts/a/dossier.json"
	if err := database.RegisterArtifactCleanup(
		ctx, committedArtifactKey, issue.ID, now.Add(-20*time.Minute),
	); err != nil {
		t.Fatal(err)
	}
	liveClaimCleanup, err := database.ClaimArtifactCleanup(
		ctx, now.Add(36*time.Second), 5*time.Minute,
	)
	if err != nil || liveClaimCleanup != nil {
		t.Fatalf("live generation artifact cleanup claim=%#v err=%v", liveClaimCleanup, err)
	}
	err = database.CompleteIssue(ctx, issue.ID, CompleteIssueInput{
		ClaimToken: claim.Token, GenerationID: uuid.NewString(),
		ArtifactKey: committedArtifactKey, Checksum: "abc", Bytes: 100,
		Title: "A generated Dossier",
		History: domain.LearningHistoryEntry{
			Date: "2026-07-19", GeneratedAt: now,
			LessonSummary: "Summary", LearningObjective: "Objective",
			Concepts:     []string{"causal mechanism", "evidence quality"},
			SourceTitles: []string{"Systems evidence review"},
			RetrievalPrompts: []domain.RetrievalPrompt{
				{
					ID: "retrieval-1", Prompt: "What is the mechanism?",
					AnswerRubric:          "Explain the mechanism and its limits.",
					CorrectiveExplanation: "Trace the mechanism step by step.",
					ConceptIDs:            []string{"mechanism", "evidence"},
				},
				{
					ID: "retrieval-2", Prompt: "What evidence supports it?",
					AnswerRubric:          "Name the relevant evidence.",
					CorrectiveExplanation: "Return to the cited evidence.",
					ConceptIDs:            []string{"evidence"},
				},
				{
					ID: "retrieval-3", Prompt: "When does it fail?",
					AnswerRubric:          "Name an important boundary condition.",
					CorrectiveExplanation: "Review the skeptical section.",
					ConceptIDs:            []string{"mechanism"},
				},
			},
			ConceptStates: []domain.LearningConcept{
				{ID: "mechanism", Label: "Mechanism", Role: "core"},
				{ID: "evidence", Label: "Evidence", Role: "core"},
			},
		},
		HistoryLimit: 14, CompletedAt: now.Add(37 * time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}
	var cleanupCount int
	if err := database.pool.QueryRow(ctx, `
		SELECT count(*) FROM artifact_cleanup_queue WHERE artifact_key = $1
	`, committedArtifactKey).Scan(&cleanupCount); err != nil || cleanupCount != 0 {
		t.Fatalf("committed artifact cleanup count=%d err=%v", cleanupCount, err)
	}
	if err := database.RegisterArtifactCleanup(
		ctx, committedArtifactKey, issue.ID, now.Add(-20*time.Minute),
	); err != nil {
		t.Fatal(err)
	}
	protectedClaim, err := database.ClaimArtifactCleanup(
		ctx, now.Add(time.Minute), 5*time.Minute,
	)
	if err != nil || protectedClaim != nil {
		t.Fatalf("referenced artifact cleanup claim=%#v err=%v", protectedClaim, err)
	}
	if err := database.CancelArtifactCleanup(ctx, committedArtifactKey); err != nil {
		t.Fatal(err)
	}
	unreferencedKey := "accounts/a/issues/orphan/generation.json.gz"
	if err := database.RegisterArtifactCleanup(
		ctx, unreferencedKey, issue.ID, now.Add(-20*time.Minute),
	); err != nil {
		t.Fatal(err)
	}
	orphanClaim, err := database.ClaimArtifactCleanup(
		ctx, now.Add(time.Minute), 5*time.Minute,
	)
	if err != nil || orphanClaim == nil || orphanClaim.Key != unreferencedKey {
		t.Fatalf("orphan artifact cleanup claim=%#v err=%v", orphanClaim, err)
	}
	if err := database.CompleteArtifactCleanup(
		ctx, orphanClaim.Key, orphanClaim.Token,
	); err != nil {
		t.Fatal(err)
	}
	deferredIssue, err := database.EnqueueManualIssue(
		ctx,
		account.ID,
		newsletter.Newsletter.ID,
		5,
	)
	if err != nil {
		t.Fatalf("enqueue deferrable Issue: %v", err)
	}
	deferredClaim, err := database.ClaimNextIssue(
		ctx,
		now.Add(38*time.Second),
		5*time.Minute,
		1,
		5,
		100,
		10_000_000,
		1_000_000,
		IssueAttemptContext{WorkerID: "test-worker", DeploymentVersion: "test"},
	)
	if err != nil || deferredClaim == nil || deferredClaim.Issue.ID != deferredIssue.ID {
		t.Fatalf("deferrable claim=%#v err=%v", deferredClaim, err)
	}
	deferCause := failure.New(
		"no_worthwhile_evidence",
		failure.CategoryInsufficientEvidence,
		"source_intelligence",
		false,
		failure.PublicNoEvidence,
		errors.New("private source diagnostic"),
	)
	if err := database.DeferIssue(
		ctx,
		deferredIssue.ID,
		deferredClaim.Token,
		deferCause,
		now.Add(39*time.Second),
	); err != nil {
		t.Fatalf("defer Issue: %v", err)
	}
	deferredProjection, err := database.GetIssue(ctx, account.ID, deferredIssue.ID)
	if err != nil {
		t.Fatal(err)
	}
	if deferredProjection.Status != domain.IssueDeferred ||
		deferredProjection.FailureCode != "no_worthwhile_evidence" ||
		deferredProjection.Error != failure.PublicNoEvidence ||
		strings.Contains(deferredProjection.Error, "private source diagnostic") {
		t.Fatalf("unsafe deferred Issue projection: %#v", deferredProjection)
	}
	var deferredAttemptStatus string
	if err := database.pool.QueryRow(ctx, `
		SELECT status FROM issue_attempts WHERE id = $1
	`, deferredClaim.Token).Scan(&deferredAttemptStatus); err != nil {
		t.Fatal(err)
	}
	if deferredAttemptStatus != "deferred" {
		t.Fatalf("deferred attempt status=%q", deferredAttemptStatus)
	}
	if err := database.RetryIssue(
		ctx,
		account.ID,
		deferredIssue.ID,
		now.Add(40*time.Second),
	); !errors.Is(err, ErrConflict) {
		t.Fatalf("deferred Issue should not retry as a failure: %v", err)
	}
	checkpoints, err = database.LoadIssueCheckpoints(ctx, issue.ID, "fingerprint-1")
	if err != nil || len(checkpoints) != 0 {
		t.Fatalf("completed Issue retained checkpoints: %#v err=%v", checkpoints, err)
	}
	history, err := database.LoadLearningHistory(ctx, newsletter.Newsletter.ID, 14)
	if err != nil || len(history) != 1 {
		t.Fatalf("history=%#v err=%v", history, err)
	}
	for _, search := range []string{
		"causal mechanism",
		"Systems evidence",
		"does it fail",
	} {
		matches, cursor, err := database.ListLibraryLessonsPage(
			ctx,
			account.ID,
			search,
			LibraryAll,
			24,
			nil,
		)
		if err != nil || len(matches) != 1 || matches[0].ID != issue.ID || cursor != nil {
			t.Fatalf(
				"Library search %q=%#v cursor=%#v err=%v",
				search,
				matches,
				cursor,
				err,
			)
		}
	}
	note, err := database.CreateLessonNote(
		ctx,
		account.ID,
		issue.ID,
		LessonNoteInput{
			Kind:       "question",
			AnchorType: "claim",
			AnchorID:   "claim-1",
			Body:       "What evidence would change this conclusion?",
			QuotedText: "A source-backed claim.",
		},
		now.Add(37*time.Second),
	)
	if err != nil || note.ID == "" {
		t.Fatalf("create Lesson Note=%#v err=%v", note, err)
	}
	notes, err := database.ListLessonNotes(ctx, account.ID, issue.ID)
	if err != nil || len(notes) != 1 || notes[0].AnchorID != "claim-1" {
		t.Fatalf("Lesson Notes=%#v err=%v", notes, err)
	}
	reviews, err := database.ListWorkspaceReviews(ctx, account.ID, 8, now.Add(37*time.Second))
	if err != nil || len(reviews) != 0 {
		t.Fatalf("reviews should wait for lesson completion: %#v err=%v", reviews, err)
	}
	retrievalDraft, err := database.SaveLessonRetrievalDraft(
		ctx,
		account.ID,
		issue.ID,
		LessonRetrievalInput{
			PromptKey: "retrieval-1",
			Response:  "The mechanism changes outcomes only within its evidence boundary.",
		},
		now.Add(37*time.Second),
	)
	if err != nil || retrievalDraft.Response == "" || retrievalDraft.RevealedAt != nil {
		t.Fatalf("Lesson Retrieval draft=%#v err=%v", retrievalDraft, err)
	}
	retrievalResponse, err := database.RevealLessonRetrieval(
		ctx,
		account.ID,
		issue.ID,
		LessonRetrievalInput{
			PromptKey: "retrieval-1",
			Response:  "The mechanism changes outcomes only within its evidence boundary.",
		},
		now.Add(37*time.Second),
	)
	if err != nil || retrievalResponse.Response == "" || retrievalResponse.Skipped {
		t.Fatalf("Lesson Retrieval response=%#v err=%v", retrievalResponse, err)
	}
	if _, err := database.RevealLessonRetrieval(
		ctx,
		account.ID,
		issue.ID,
		LessonRetrievalInput{PromptKey: "retrieval-2", Skipped: true},
		now.Add(37*time.Second),
	); err != nil {
		t.Fatalf("skip Lesson Retrieval: %v", err)
	}
	if _, err := database.RevealLessonRetrieval(
		ctx,
		account.ID,
		issue.ID,
		LessonRetrievalInput{PromptKey: "retrieval-1", Response: "A changed answer."},
		now.Add(37*time.Second),
	); !errors.Is(err, ErrConflict) {
		t.Fatalf("revealed response was mutable: %v", err)
	}
	retrievalResponses, err := database.ListLessonRetrievalResponses(
		ctx, account.ID, issue.ID,
	)
	if err != nil || len(retrievalResponses) != 2 ||
		!retrievalResponses[1].Skipped {
		t.Fatalf("Lesson Retrieval responses=%#v err=%v", retrievalResponses, err)
	}
	var firstCycleEvents int
	if err := database.pool.QueryRow(ctx, `
		SELECT count(*) FROM product_events
		WHERE account_id = $1
		  AND event_name IN ('first_retrieval_completed', 'activation_completed')
	`, account.ID).Scan(&firstCycleEvents); err != nil || firstCycleEvents != 2 {
		t.Fatalf("first-cycle events=%d err=%v", firstCycleEvents, err)
	}
	partialProgress, err := database.SaveLessonProgress(
		ctx,
		account.ID,
		issue.ID,
		37,
		now.Add(37500*time.Millisecond),
	)
	if err != nil || partialProgress.Progress != 37 || partialProgress.CompletedAt != nil {
		t.Fatalf("partial lesson progress=%#v err=%v", partialProgress, err)
	}
	progressSnapshot, err := database.GetLessonProgress(ctx, account.ID, issue.ID)
	if err != nil || progressSnapshot == nil || progressSnapshot.Progress != 37 {
		t.Fatalf("Lesson Progress snapshot=%#v err=%v", progressSnapshot, err)
	}
	todayFocus, err := database.RefreshTodayFocus(
		ctx, account.ID, now.Add(37500*time.Millisecond),
	)
	if err != nil || todayFocus.Kind != "lesson" || todayFocus.SubjectID != issue.ID ||
		todayFocus.ReasonCode != "continue_in_progress" || todayFocus.Progress != 37 {
		t.Fatalf("in-progress Today focus=%#v err=%v", todayFocus, err)
	}
	var storedReason string
	if err := database.pool.QueryRow(ctx, `
		SELECT reason_text FROM today_focus_selections WHERE account_id = $1
	`, account.ID).Scan(&storedReason); err != nil || storedReason == "" {
		t.Fatalf("stored Today reason=%q err=%v", storedReason, err)
	}
	repeatedFocus, err := database.RefreshTodayFocus(
		ctx, account.ID, now.Add(38500*time.Millisecond),
	)
	if err != nil || repeatedFocus.SubjectID != todayFocus.SubjectID ||
		!repeatedFocus.SelectedAt.Equal(todayFocus.SelectedAt) {
		t.Fatalf("stable Today focus=%#v first=%#v err=%v", repeatedFocus, todayFocus, err)
	}
	navigation, err := database.GetLessonNavigation(ctx, account.ID, issue.ID)
	if err != nil || navigation.Previous != nil || navigation.Next != nil ||
		navigation.NextReviewAt != nil {
		t.Fatalf("pre-completion Lesson navigation=%#v err=%v", navigation, err)
	}
	library, libraryCursor, err := database.ListLibraryLessonsPage(
		ctx,
		account.ID,
		"generated dossier",
		LibraryInProgress,
		24,
		nil,
	)
	if err != nil || len(library) != 1 || library[0].ID != issue.ID ||
		library[0].Progress == nil || library[0].Progress.Progress != 37 ||
		libraryCursor != nil {
		t.Fatalf("Library lessons=%#v cursor=%#v err=%v", library, libraryCursor, err)
	}
	completedLesson, err := database.CompleteLesson(
		ctx,
		account.ID,
		issue.ID,
		now.Add(38*time.Second),
	)
	if err != nil || completedLesson.Progress != 100 || completedLesson.CompletedAt == nil {
		t.Fatalf("completed lesson=%#v err=%v", completedLesson, err)
	}
	navigation, err = database.GetLessonNavigation(ctx, account.ID, issue.ID)
	reviewDue := now.Add(38*time.Second + 24*time.Hour)
	if err != nil || navigation.NextReviewAt == nil || !navigation.NextReviewAt.Equal(reviewDue) {
		t.Fatalf("completed Lesson navigation=%#v err=%v", navigation, err)
	}
	reviews, err = database.ListWorkspaceReviews(ctx, account.ID, 8, now.Add(38*time.Second))
	if err != nil || len(reviews) != 0 {
		t.Fatalf("reviews should be spaced after in-lesson retrieval: %#v err=%v", reviews, err)
	}
	todayFocus, err = database.RefreshTodayFocus(ctx, account.ID, now.Add(38*time.Second))
	if err != nil || todayFocus.Kind == "review" {
		t.Fatalf("review surfaced without spacing: focus=%#v err=%v", todayFocus, err)
	}
	reviews, err = database.ListWorkspaceReviews(ctx, account.ID, 8, reviewDue)
	if err != nil || len(reviews) != 3 || reviews[0].IssueID != issue.ID {
		t.Fatalf("spaced workspace reviews=%#v err=%v", reviews, err)
	}
	todayFocus, err = database.RefreshTodayFocus(ctx, account.ID, reviewDue)
	if err != nil || todayFocus.Kind != "review" || todayFocus.DueCount != 3 ||
		todayFocus.ActionURL != "/review" {
		t.Fatalf("spaced review Today focus=%#v err=%v", todayFocus, err)
	}
	assessmentKey := uuid.NewString()
	assessmentAt := reviewDue.Add(time.Second)
	assessed, err := database.AssessReview(
		ctx,
		account.ID,
		reviews[0].ID,
		assessmentKey,
		ReviewSolid,
		assessmentAt,
	)
	if err != nil || assessed.Stage != 2 ||
		!assessed.DueAt.Equal(assessmentAt.Add(7*24*time.Hour)) {
		t.Fatalf("assessed review=%#v err=%v", assessed, err)
	}
	concepts, err := database.ListLearnerConcepts(
		ctx,
		account.ID,
		newsletter.Newsletter.ID,
		10,
	)
	if err != nil || len(concepts) != 2 ||
		concepts[0].CompletedCount != 1 ||
		concepts[0].ConfidenceScore != 85 {
		t.Fatalf("Learner Concepts=%#v err=%v", concepts, err)
	}
	curriculum, err := database.GetCurriculum(
		ctx,
		account.ID,
		newsletter.Newsletter.ID,
	)
	if err != nil || len(curriculum.Timeline) != 1 ||
		curriculum.Timeline[0].IssueID != issue.ID ||
		len(curriculum.Timeline[0].Concepts) != 2 {
		t.Fatalf("Curriculum timeline=%#v err=%v", curriculum, err)
	}
	if _, err := database.AssessReview(
		ctx,
		account.ID,
		reviews[0].ID,
		assessmentKey,
		ReviewSolid,
		now.Add(40*time.Second),
	); err != nil {
		t.Fatalf("idempotent Review Attempt failed: %v", err)
	}
	var attemptCount int
	if err := database.pool.QueryRow(ctx, `
		SELECT count(*) FROM review_attempts
		WHERE account_id = $1 AND idempotency_key = $2
	`, account.ID, assessmentKey).Scan(&attemptCount); err != nil || attemptCount != 1 {
		t.Fatalf("Review Attempt count=%d err=%v", attemptCount, err)
	}
	var activationEvents int
	if err := database.pool.QueryRow(ctx, `
		SELECT count(*) FROM product_events
		WHERE account_id = $1
		  AND event_name IN ('first_retrieval_completed', 'activation_completed')
	`, account.ID).Scan(&activationEvents); err != nil || activationEvents != 2 {
		t.Fatalf("activation events=%d err=%v", activationEvents, err)
	}
	difficulty := "right"
	relevance := "very_relevant"
	recall := "medium"
	feedback, err := database.SaveLessonFeedback(
		ctx,
		account.ID,
		issue.ID,
		LessonFeedbackInput{
			Difficulty:       &difficulty,
			Relevance:        &relevance,
			RecallConfidence: &recall,
		},
		now.Add(38*time.Second),
	)
	if err != nil || feedback.Difficulty == nil || *feedback.Difficulty != difficulty {
		t.Fatalf("lesson feedback=%#v err=%v", feedback, err)
	}
	feedbackSnapshot, err := database.GetLessonFeedback(ctx, account.ID, issue.ID)
	if err != nil || feedbackSnapshot == nil ||
		feedbackSnapshot.RecallConfidence == nil ||
		*feedbackSnapshot.RecallConfidence != recall {
		t.Fatalf("lesson feedback snapshot=%#v err=%v", feedbackSnapshot, err)
	}
	learnerState, err := database.LoadLearnerState(
		ctx,
		account.ID,
		newsletter.Newsletter.ID,
		10,
	)
	if err != nil || len(learnerState.Concepts) != 2 ||
		learnerState.Concepts[0].ConfidenceScore != 85 ||
		learnerState.Difficulty != difficulty ||
		learnerState.Relevance != relevance ||
		learnerState.RecallConfidence != recall ||
		len(learnerState.OpenQuestions) != 1 ||
		!strings.Contains(learnerState.OpenQuestions[0], "change this conclusion") {
		t.Fatalf("learner state=%#v err=%v", learnerState, err)
	}
	if err := database.RecordOwnedLessonEvent(
		ctx,
		account.ID,
		issue.ID,
		ProductEventLessonOpened,
		now.Add(38*time.Second),
	); err != nil {
		t.Fatalf("record lesson opened: %v", err)
	}
	retention, err := database.GetRetentionState(
		ctx,
		account.ID,
		now.Add(9*24*time.Hour),
	)
	if err != nil || retention.ActivatedAt == nil || !retention.Inactive ||
		retention.ActionURL == "" || retention.ReturnedAfterSevenDays {
		t.Fatalf("inactive retention=%#v err=%v", retention, err)
	}
	if err := database.RecordProductEvent(
		ctx,
		account.ID,
		ProductEventReviewAttempted,
		"review",
		"return-review",
		now.Add(9*24*time.Hour),
	); err != nil {
		t.Fatalf("record seven-day return: %v", err)
	}
	retention, err = database.GetRetentionState(
		ctx,
		account.ID,
		now.Add(9*24*time.Hour+time.Minute),
	)
	if err != nil || !retention.ReturnedAfterSevenDays || retention.Inactive {
		t.Fatalf("returned retention=%#v err=%v", retention, err)
	}
	var eventCount int
	if err := database.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM product_events
		WHERE account_id = $1
		  AND event_name IN (
		    'signup_completed', 'stream_created', 'lesson_generated',
		    'lesson_opened', 'lesson_completed'
		  )
	`, account.ID).Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if eventCount != 5 {
		t.Fatalf("activation milestone count=%d, want 5", eventCount)
	}
	preferences, err := database.UpdateNotificationPreferences(
		ctx,
		account.ID,
		true,
		true,
		"UTC",
		now.Add(40*time.Second),
	)
	if err != nil || !preferences.WeeklyRecap || !preferences.ReentryReminder {
		t.Fatalf("notification preferences=%#v err=%v", preferences, err)
	}
	completedAt := now.Add(38 * time.Second)
	dispatchAt := time.Date(
		completedAt.UTC().Year(),
		completedAt.UTC().Month(),
		completedAt.UTC().Day(),
		9, 0, 0, 0,
		time.UTC,
	)
	daysUntilMonday := (8 - int(dispatchAt.Weekday())) % 7
	if daysUntilMonday == 0 {
		daysUntilMonday = 7
	}
	dispatchAt = dispatchAt.AddDate(0, 0, daysUntilMonday)
	dispatchedRecaps, err := database.DispatchWeeklyRecaps(ctx, dispatchAt, 10)
	if err != nil || dispatchedRecaps != 1 {
		t.Fatalf("dispatched Recaps=%d err=%v", dispatchedRecaps, err)
	}
	dispatchedRecaps, err = database.DispatchWeeklyRecaps(ctx, dispatchAt, 10)
	if err != nil || dispatchedRecaps != 0 {
		t.Fatalf("duplicate Recaps=%d err=%v", dispatchedRecaps, err)
	}
	recapClaim, err := database.ClaimNextWeeklyRecap(
		ctx,
		dispatchAt,
		5*time.Minute,
		6,
	)
	if err != nil || recapClaim == nil ||
		recapClaim.Payload.LessonsCompleted != 1 ||
		len(recapClaim.Payload.Concepts) == 0 {
		t.Fatalf("weekly Recap claim=%#v err=%v", recapClaim, err)
	}
	if err := database.CompleteWeeklyRecap(
		ctx,
		recapClaim.ID,
		recapClaim.Token,
		"resend-recap-test",
		dispatchAt.Add(time.Second),
	); err != nil {
		t.Fatal(err)
	}
	progress, err := database.ListLessonProgress(ctx, account.ID)
	if err != nil || len(progress) != 1 || progress[0].IssueID != issue.ID {
		t.Fatalf("lesson progress=%#v err=%v", progress, err)
	}
	if _, err := database.SetIssuePublication(
		ctx,
		account.ID,
		issue.ID,
		PublicationChange{
			State: domain.PublicationPublished, AudienceConfirmed: true,
			Now: dispatchAt.Add(2 * time.Second),
		},
	); err != nil {
		t.Fatalf("publish lifecycle lesson: %v", err)
	}
	publicIssues, err := database.ListPublicIssues(ctx, site.Username, "", 10)
	if err != nil || len(publicIssues) != 1 {
		t.Fatalf("publicIssues=%#v err=%v", publicIssues, err)
	}
	historyBeforeModeration := 0
	if err := database.pool.QueryRow(
		ctx,
		"SELECT count(*) FROM learning_history WHERE issue_id = $1",
		issue.ID,
	).Scan(&historyBeforeModeration); err != nil {
		t.Fatal(err)
	}
	reportID, err := database.CreatePublicContentReport(
		ctx,
		site.Username,
		publicIssues[0].PublicID,
		"citation",
		"The cited source does not support this sentence.",
		strings.Repeat("f", 64),
		now.Add(41*time.Second),
	)
	if err != nil || reportID == "" {
		t.Fatalf("create public content report id=%q err=%v", reportID, err)
	}
	correction, err := database.AddPublicCorrection(
		ctx,
		account.ID,
		issue.ID,
		"The source supports a narrower claim; the public wording has been corrected here.",
		now.Add(42*time.Second),
	)
	if err != nil {
		t.Fatalf("add public correction: %v", err)
	}
	if err := database.SetIssueModerationState(
		ctx,
		account.ID,
		issue.ID,
		"held",
		"Reviewing the citation report.",
		now.Add(43*time.Second),
	); err != nil {
		t.Fatalf("hold public issue: %v", err)
	}
	publicIssues, err = database.ListPublicIssues(ctx, site.Username, "", 10)
	if err != nil || len(publicIssues) != 0 {
		t.Fatalf("held issue remained index eligible: %#v %v", publicIssues, err)
	}
	if err := database.ResolvePublicContentReport(
		ctx,
		account.ID,
		reportID,
		"resolved",
		"Published a correction and verified the cited source.",
		now.Add(44*time.Second),
	); err != nil {
		t.Fatalf("resolve public content report: %v", err)
	}
	if err := database.SetIssueModerationState(
		ctx,
		account.ID,
		issue.ID,
		"clear",
		"Correction published and source verified.",
		now.Add(45*time.Second),
	); err != nil {
		t.Fatalf("clear public issue: %v", err)
	}
	moderation, err := database.GetIssueModeration(ctx, account.ID, issue.ID)
	if err != nil || moderation.State != "clear" ||
		len(moderation.Corrections) != 1 || len(moderation.Reports) != 1 ||
		len(moderation.Actions) < 4 {
		t.Fatalf("issue moderation=%#v err=%v", moderation, err)
	}
	historyAfterModeration := 0
	if err := database.pool.QueryRow(
		ctx,
		"SELECT count(*) FROM learning_history WHERE issue_id = $1",
		issue.ID,
	).Scan(&historyAfterModeration); err != nil {
		t.Fatal(err)
	}
	if historyAfterModeration != historyBeforeModeration {
		t.Fatalf(
			"public correction mutated private history: before=%d after=%d",
			historyBeforeModeration,
			historyAfterModeration,
		)
	}
	if err := database.RetractPublicCorrection(
		ctx,
		account.ID,
		correction.ID,
		now.Add(46*time.Second),
	); err != nil {
		t.Fatalf("retract public correction: %v", err)
	}
	publicIssues, err = database.ListPublicIssues(ctx, site.Username, "", 10)
	if err != nil || len(publicIssues) != 1 {
		t.Fatalf("cleared issue did not regain eligibility: %#v %v", publicIssues, err)
	}
	deliveryClaim, err := database.ClaimNextDelivery(
		ctx,
		now.Add(38*time.Second),
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
		now.Add(39*time.Second),
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
	operations, err := database.OperationalSnapshot(ctx, time.Now().UTC())
	if err != nil || operations.QueuedIssues < 1 ||
		operations.DatabaseMax < 1 || operations.DatabaseTotal < 1 ||
		operations.ModelInputTokens < 100 ||
		operations.ModelOutputTokens < 50 ||
		operations.ModelProviderRetries < 1 ||
		operations.ModelCostMicroUSD < 200 {
		t.Fatalf("operational snapshot=%#v err=%v", operations, err)
	}
	expiredDeliveryToken := uuid.New()
	expiredDeliveryAt := now.Add(40 * time.Second)
	if _, err := database.pool.Exec(ctx, `
		INSERT INTO delivery_receipts (
			issue_id, status, attempt_count, available_at,
			claim_token, claim_expires_at, created_at, updated_at
		)
		VALUES ($1, 'delivering', 1, $2, $3, $2, $2, $2)
	`, newerIssue.ID, expiredDeliveryAt, expiredDeliveryToken); err != nil {
		t.Fatal(err)
	}
	recovered, err = database.RecoverExpiredClaims(
		ctx,
		expiredDeliveryAt.Add(time.Second),
		3,
		6,
	)
	if err != nil || recovered != 1 {
		t.Fatalf("recover expired Delivery Claim: recovered=%d err=%v", recovered, err)
	}
	receipt, err := database.GetDelivery(ctx, account.ID, newerIssue.ID)
	if err != nil || receipt == nil || receipt.Status != domain.DeliveryUnknown {
		t.Fatalf("expired Delivery outcome=%#v err=%v", receipt, err)
	}
	page, cursor, err := database.ListWorkspaceIssuesPage(ctx, account.ID, 1, nil)
	if err != nil || len(page) != 1 || page[0].ID != newerIssue.ID || cursor == nil {
		t.Fatalf("first workspace page=%#v cursor=%#v err=%v", page, cursor, err)
	}
	older, finalCursor, err := database.ListWorkspaceIssuesPage(ctx, account.ID, 1, cursor)
	if err != nil || len(older) != 1 || older[0].ID != deferredIssue.ID ||
		older[0].Status != domain.IssueDeferred || finalCursor == nil {
		t.Fatalf("second workspace page=%#v cursor=%#v err=%v", older, finalCursor, err)
	}
	oldest, exhaustedCursor, err := database.ListWorkspaceIssuesPage(
		ctx,
		account.ID,
		1,
		finalCursor,
	)
	if err != nil || len(oldest) != 1 || oldest[0].ID != issue.ID || exhaustedCursor != nil {
		t.Fatalf("third workspace page=%#v cursor=%#v err=%v", oldest, exhaustedCursor, err)
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
	if _, err := database.CreateLessonNote(
		ctx,
		other.ID,
		issue.ID,
		LessonNoteInput{
			Kind: "note", AnchorType: "lesson", Body: "unauthorized",
		},
		now,
	); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-Account Lesson Note create was not denied: %v", err)
	}
	if err := database.DeleteLessonNote(ctx, other.ID, note.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-Account Lesson Note delete was not denied: %v", err)
	}
	if err := database.DeleteLessonNote(ctx, account.ID, note.ID); err != nil {
		t.Fatalf("delete Lesson Note: %v", err)
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
	erasedAt := time.Now().UTC()
	if err := database.CompleteAccountDeletion(
		ctx,
		deletion.AccountID,
		deletion.Token,
		erasedAt,
	); err != nil {
		t.Fatalf("complete Account erasure: %v", err)
	}
	var accountsRemaining, newslettersRemaining, receipts int
	if err := database.pool.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM accounts WHERE id = $1),
		  (SELECT count(*) FROM newsletters WHERE owner_account_id = $1),
		  (SELECT count(*) FROM privacy_erasure_receipts
		   WHERE account_fingerprint = $2)
	`, account.ID, identityFingerprint(account.ID)).Scan(
		&accountsRemaining,
		&newslettersRemaining,
		&receipts,
	); err != nil {
		t.Fatal(err)
	}
	if accountsRemaining != 0 || newslettersRemaining != 0 || receipts != 1 {
		t.Fatalf(
			"erasure result accounts=%d newsletters=%d receipts=%d",
			accountsRemaining,
			newslettersRemaining,
			receipts,
		)
	}
	postErasure, err := database.SyncAccountIdentity(
		ctx,
		account.ClerkUserID,
		"learner@example.com",
		domain.AccountActive,
		identityTime+20,
	)
	if err != nil || postErasure.Status != domain.AccountDeleted {
		t.Fatalf("deleted identity was recreated after erasure: %#v %v", postErasure, err)
	}
	if _, err := database.EnsureAccount(ctx, account.ClerkUserID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("deleted identity regained authenticated access after erasure: %v", err)
	}
}

func TestOperationalSnapshotCountsConsecutiveAccountFailures(t *testing.T) {
	database := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	now := time.Now().UTC()
	account, err := database.SyncAccountIdentity(
		ctx,
		"clerk-alerts-"+uuid.NewString(),
		"alerts@example.com",
		domain.AccountActive,
		now.UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	requirePaidIntegrationPlan(t, ctx, database, account.ID, now)
	input := integrationNewsletterInput(nil)
	input.SourceMode = domain.SourceModeDiscovered
	created, err := database.CreateNewsletter(
		ctx,
		account.ID,
		input,
		10,
		5,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.pool.Exec(ctx, `
		UPDATE issues SET status = 'failed', completed_at = $2
		WHERE id = $1
	`, created.FirstIssue.ID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := database.pool.Exec(ctx, `
		INSERT INTO issues (
		  id, newsletter_id, trigger, status, available_at, public_id,
		  publication_state, created_at, completed_at
		)
		VALUES ($1, $2, 'manual', 'failed', $3, $4, 'private', $3, $3)
	`, uuid.New(), created.Newsletter.ID, now.Add(time.Second), uuid.New()); err != nil {
		t.Fatal(err)
	}
	snapshot, err := database.OperationalSnapshot(ctx, now.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.AccountsConsecutiveFailures < 1 {
		t.Fatalf("consecutive failure accounts=%d", snapshot.AccountsConsecutiveFailures)
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
	requirePaidIntegrationPlan(t, ctx, database, account.ID, time.Now().UTC())
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
		t.Fatalf("projected sources=%#v, want %#v", updated.Sources, input.Sources)
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

func TestSourceControlsPreserveFrozenEvidenceIntegration(t *testing.T) {
	database := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	now := time.Now().UTC()
	account, err := database.SyncAccountIdentity(
		ctx,
		"clerk-source-controls-"+uuid.NewString(),
		"source-controls@example.com",
		domain.AccountActive,
		now.UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	requirePaidIntegrationPlan(t, ctx, database, account.ID, now)
	input := integrationNewsletterInput([]domain.SourceDefinition{{
		Name: "Original", URL: "https://example.com/original", Limit: 8,
	}})
	created, err := database.CreateNewsletter(ctx, account.ID, input, 10, 5)
	if err != nil {
		t.Fatal(err)
	}
	var originalID string
	if err := database.pool.QueryRow(ctx, `
		SELECT id::text FROM source_specs
		WHERE newsletter_id = $1 AND canonical_url = $2
	`, created.Newsletter.ID, input.Sources[0].URL).Scan(&originalID); err != nil {
		t.Fatal(err)
	}
	endpoint, err := database.UpsertSourceEndpoint(ctx, domain.SourceEndpoint{
		ID: uuid.NewString(), SourceSpecID: originalID,
		EndpointURL: input.Sources[0].URL, CanonicalURL: input.Sources[0].URL,
		Kind: domain.SourceKindHTML, Health: "healthy", UpdatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshotID, err := database.InsertSourceSnapshot(ctx, domain.SourceSnapshot{
		ID: uuid.NewString(), SourceEndpointID: endpoint.ID,
		ItemKey: input.Sources[0].URL, Title: "Frozen original",
		CanonicalURL: input.Sources[0].URL, Content: strings.Repeat("evidence ", 80),
		ContentSource: "article", ContentSHA256: "frozen-original",
		Metadata: `{}`, FetchedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if inserted, err := database.InsertIssueSources(ctx, created.FirstIssue.ID, []domain.IssueSource{{
		IssueID: created.FirstIssue.ID, SourceSnapshotID: snapshotID,
		Position: 0, CreatedAt: now,
	}}); err != nil || !inserted {
		t.Fatalf("freeze inserted=%v err=%v", inserted, err)
	}
	replacementID, err := database.ReplaceProvidedSource(
		ctx,
		account.ID,
		created.Newsletter.ID,
		originalID,
		domain.SourceDefinition{
			Name: "Replacement", URL: "https://example.net/replacement", Limit: 8,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	frozen, err := database.GetIssueSources(ctx, created.FirstIssue.ID)
	if err != nil || len(frozen) != 1 || frozen[0].ID != snapshotID ||
		frozen[0].CanonicalURL != input.Sources[0].URL {
		t.Fatalf("replacement mutated frozen evidence: %#v err=%v", frozen, err)
	}
	var originalState, replacementState string
	if err := database.pool.QueryRow(ctx, `
		SELECT
		  (SELECT state FROM source_specs WHERE id = $1),
		  (SELECT state FROM source_specs WHERE id = $2)
	`, originalID, replacementID).Scan(&originalState, &replacementState); err != nil {
		t.Fatal(err)
	}
	if originalState != "disabled" || replacementState != "active" {
		t.Fatalf("source states original=%s replacement=%s", originalState, replacementState)
	}
	replacedRecord, err := database.GetNewsletter(ctx, account.ID, created.Newsletter.ID)
	if err != nil || len(replacedRecord.Sources) != 1 ||
		replacedRecord.Sources[0].URL != "https://example.net/replacement" {
		t.Fatalf("replacement projection=%#v err=%v", replacedRecord.Sources, err)
	}
	if err := database.SetSourcePreference(
		ctx,
		account.ID,
		created.Newsletter.ID,
		replacementID,
		domain.SourcePreferencePreferred,
	); err != nil {
		t.Fatal(err)
	}
	active, err := database.ListActiveSourceSpecs(ctx, created.Newsletter.ID)
	if err != nil || len(active) != 1 || active[0].Preference != domain.SourcePreferencePreferred {
		t.Fatalf("preferred active sources=%#v err=%v", active, err)
	}
	if err := database.SetSourcePreference(
		ctx,
		account.ID,
		created.Newsletter.ID,
		replacementID,
		domain.SourcePreferenceBlocked,
	); !errors.Is(err, ErrConflict) {
		t.Fatalf("only provided source was blockable: %v", err)
	}
	if err := database.SetNewsletterSourceMode(
		ctx,
		account.ID,
		created.Newsletter.ID,
		domain.SourceModeHybrid,
	); err != nil {
		t.Fatal(err)
	}
	if err := database.SetSourcePreference(
		ctx,
		account.ID,
		created.Newsletter.ID,
		replacementID,
		domain.SourcePreferenceBlocked,
	); err != nil {
		t.Fatal(err)
	}
	record, err := database.GetNewsletter(ctx, account.ID, created.Newsletter.ID)
	if err != nil || record.SourceMode != domain.SourceModeDiscovered {
		t.Fatalf("last hybrid source did not fall back to discovery: %#v err=%v", record, err)
	}
	if err := database.SetNewsletterSourceMode(
		ctx,
		account.ID,
		created.Newsletter.ID,
		domain.SourceModeProvided,
	); !errors.Is(err, ErrConflict) {
		t.Fatalf("provided mode accepted only blocked sources: %v", err)
	}
	other, err := database.SyncAccountIdentity(
		ctx,
		"clerk-source-controls-other-"+uuid.NewString(),
		"source-controls-other@example.com",
		domain.AccountActive,
		now.UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.SetSourcePreference(
		ctx,
		other.ID,
		created.Newsletter.ID,
		replacementID,
		domain.SourcePreferenceNeutral,
	); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-Account source preference was not denied: %v", err)
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
	requirePaidIntegrationPlan(t, ctx, database, account.ID, time.Now().UTC())
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

func TestSourcePortfolioApprovalLifecycleIntegration(t *testing.T) {
	database := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	now := time.Now().UTC()
	account, err := database.SyncAccountIdentity(
		ctx,
		"clerk-source-approval-"+uuid.NewString(),
		"source-approval@example.com",
		domain.AccountActive,
		now.UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	requirePaidIntegrationPlan(t, ctx, database, account.ID, now)
	input := integrationNewsletterInput(nil)
	input.SourceMode = domain.SourceModeDiscovered
	input.SourceReviewMode = domain.SourceReviewBeforeLesson
	created, err := database.CreateNewsletter(ctx, account.ID, input, 10, 5)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.pool.Exec(ctx, `
		UPDATE issues SET status = 'cancelled', completed_at = $2
		WHERE status = 'queued' AND id <> $1::uuid
	`, created.FirstIssue.ID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := database.pool.Exec(ctx, `
		UPDATE issues SET available_at = $2, completed_at = NULL WHERE id = $1::uuid
	`, created.FirstIssue.ID, now.Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	claim, err := database.ClaimNextIssue(
		ctx, now, 5*time.Minute, 1, 5, 100,
		10_000_000, 1_000_000,
		IssueAttemptContext{WorkerID: "approval-test", DeploymentVersion: "test"},
	)
	if err != nil || claim == nil || claim.Issue.ID != created.FirstIssue.ID {
		t.Fatalf("claim=%#v err=%v", claim, err)
	}
	if err := database.AwaitSourceApproval(
		ctx,
		claim.Issue.ID,
		claim.Token,
		now.Add(time.Second),
	); err != nil {
		t.Fatal(err)
	}
	var issueStatus, attemptStatus string
	var attemptCount int
	if err := database.pool.QueryRow(ctx, `
		SELECT i.status, i.attempt_count, a.status
		FROM issues i JOIN issue_attempts a ON a.id = $2::uuid
		WHERE i.id = $1::uuid
	`, claim.Issue.ID, claim.Token).Scan(
		&issueStatus,
		&attemptCount,
		&attemptStatus,
	); err != nil {
		t.Fatal(err)
	}
	if issueStatus != "awaiting_approval" || attemptStatus != "awaiting_approval" || attemptCount != 0 {
		t.Fatalf("issue=%s attempt=%s count=%d", issueStatus, attemptStatus, attemptCount)
	}
	if err := database.ApproveSourcePortfolio(
		ctx,
		uuid.NewString(),
		created.Newsletter.ID,
		now.Add(2*time.Second),
	); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-account approval err=%v", err)
	}
	if err := database.ApproveSourcePortfolio(
		ctx,
		account.ID,
		created.Newsletter.ID,
		now.Add(2*time.Second),
	); err != nil {
		t.Fatal(err)
	}
	var approvedAt *time.Time
	if err := database.pool.QueryRow(ctx, `
		SELECT i.status, n.source_approved_at
		FROM issues i JOIN newsletters n ON n.id = i.newsletter_id
		WHERE i.id = $1::uuid
	`, claim.Issue.ID).Scan(&issueStatus, &approvedAt); err != nil {
		t.Fatal(err)
	}
	if issueStatus != "queued" || approvedAt == nil {
		t.Fatalf("approved issue=%s approvedAt=%v", issueStatus, approvedAt)
	}
	reclaimed, err := database.ClaimNextIssue(
		ctx, now.Add(3*time.Second), 5*time.Minute, 1, 5, 100,
		10_000_000, 1_000_000,
		IssueAttemptContext{WorkerID: "approval-test", DeploymentVersion: "test"},
	)
	if err != nil || reclaimed == nil || reclaimed.Issue.ID != claim.Issue.ID {
		t.Fatalf("approved portfolio was not re-queued: claim=%#v err=%v", reclaimed, err)
	}
}

func TestLearningRhythmSchedulingAndBacklogThrottleIntegration(t *testing.T) {
	database := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	now := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	account, err := database.SyncAccountIdentity(
		ctx,
		"clerk-rhythm-"+uuid.NewString(),
		"rhythm@example.com",
		domain.AccountActive,
		now.UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	requirePaidIntegrationPlan(t, ctx, database, account.ID, now)
	created, err := database.CreateNewsletter(
		ctx,
		account.ID,
		integrationNewsletterInput([]domain.SourceDefinition{{
			Name: "Evidence", URL: "https://example.com/rhythm", Limit: 5,
		}}),
		10,
		20,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.pool.Exec(ctx, `
		UPDATE issues SET status = 'cancelled', completed_at = $2
		WHERE id = $1
	`, created.FirstIssue.ID, now); err != nil {
		t.Fatal(err)
	}
	record, err := database.SetNewsletterRhythm(
		ctx,
		account.ID,
		created.Newsletter.ID,
		RhythmInput{
			Mode: domain.RhythmSelectedWeekdays, SelectedWeekdays: []int{1, 5},
			AutoThrottleEnabled: false, UnopenedLessonLimit: 3,
		},
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if record.RhythmMode != domain.RhythmSelectedWeekdays ||
		record.EffectiveRhythmMode != domain.RhythmSelectedWeekdays ||
		!reflect.DeepEqual(record.SelectedWeekdays, []int{1, 5}) ||
		!record.NextRunAt.Equal(time.Date(2026, 8, 14, 8, 0, 0, 0, time.UTC)) {
		t.Fatalf("selected weekday rhythm=%#v", record.Newsletter)
	}
	other, err := database.SyncAccountIdentity(
		ctx,
		"clerk-rhythm-other-"+uuid.NewString(),
		"rhythm-other@example.com",
		domain.AccountActive,
		now.UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.SetNewsletterRhythm(
		ctx,
		other.ID,
		created.Newsletter.ID,
		RhythmInput{Mode: domain.RhythmDaily, UnopenedLessonLimit: 3},
		now,
	); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-account rhythm update err=%v", err)
	}
	if _, err := database.SetNewsletterRhythm(
		ctx,
		account.ID,
		created.Newsletter.ID,
		RhythmInput{
			Mode: domain.RhythmWeeklySynthesis, SelectedWeekdays: []int{4},
			AutoThrottleEnabled: false, UnopenedLessonLimit: 3,
		},
		now,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := database.pool.Exec(ctx, `
		UPDATE newsletters SET next_run_at = $2 WHERE id = $1
	`, created.Newsletter.ID, now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	dispatched, err := database.DispatchDue(ctx, now, 10)
	if err != nil || dispatched != 1 {
		t.Fatalf("weekly dispatch=%d err=%v", dispatched, err)
	}
	var requestedType string
	if err := database.pool.QueryRow(ctx, `
		SELECT requested_lesson_type
		FROM issues
		WHERE newsletter_id = $1 AND trigger = 'scheduled'
		ORDER BY created_at DESC LIMIT 1
	`, created.Newsletter.ID).Scan(&requestedType); err != nil {
		t.Fatal(err)
	}
	if requestedType != string(domain.LessonSynthesis) {
		t.Fatalf("requested lesson type=%q", requestedType)
	}
	if _, err := database.pool.Exec(ctx, `
		INSERT INTO issues (
			id, newsletter_id, trigger, status, available_at, public_id,
			publication_state, generation_id, artifact_key, created_at, completed_at
		)
		SELECT gen_random_uuid(), $1, 'manual', 'generated', $2,
		       gen_random_uuid(), 'private', gen_random_uuid(),
		       'test/rhythm/' || value::text, $2, $2
		FROM generate_series(1, 3) AS value
	`, created.Newsletter.ID, now); err != nil {
		t.Fatal(err)
	}
	record, err = database.SetNewsletterRhythm(
		ctx,
		account.ID,
		created.Newsletter.ID,
		RhythmInput{
			Mode: domain.RhythmDaily, SelectedWeekdays: []int{1, 2, 3, 4, 5},
			AutoThrottleEnabled: true, UnopenedLessonLimit: 3,
		},
		now.Add(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	if record.EffectiveRhythmMode != domain.RhythmWeeklySynthesis ||
		record.RhythmThrottledAt == nil || !strings.Contains(record.RhythmReason, "3 lessons") {
		t.Fatalf("throttled rhythm=%#v", record.Newsletter)
	}
	throttleDue := now.Add(8 * 24 * time.Hour)
	if _, err := database.pool.Exec(ctx, `
		UPDATE newsletters SET next_run_at = $2 WHERE id = $1
	`, created.Newsletter.ID, throttleDue.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := database.DispatchDue(ctx, throttleDue, 10); err != nil {
		t.Fatalf("backlog dispatch err=%v", err)
	}
	var throttledScheduled int
	if err := database.pool.QueryRow(ctx, `
		SELECT count(*) FROM issues
		WHERE newsletter_id = $1 AND trigger = 'scheduled'
		  AND created_at >= $2
	`, created.Newsletter.ID, throttleDue.Add(-time.Hour)).Scan(&throttledScheduled); err != nil {
		t.Fatal(err)
	}
	if throttledScheduled != 0 {
		t.Fatalf("throttled newsletter dispatched %d scheduled issues", throttledScheduled)
	}
	if _, err := database.pool.Exec(ctx, `
		INSERT INTO product_events (
			account_id, event_name, subject_type, subject_id, occurred_at, created_at
		)
		SELECT $1, 'lesson_opened', 'lesson', i.id::text, $3, $3
		FROM issues i
		WHERE i.newsletter_id = $2 AND i.status = 'generated'
		ON CONFLICT (account_id, event_name, subject_id) DO NOTHING
	`, account.ID, created.Newsletter.ID, throttleDue); err != nil {
		t.Fatal(err)
	}
	recoveryDue := throttleDue.Add(8 * 24 * time.Hour)
	if _, err := database.pool.Exec(ctx, `
		UPDATE newsletters SET next_run_at = $2 WHERE id = $1
	`, created.Newsletter.ID, recoveryDue.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := database.DispatchDue(ctx, recoveryDue, 10); err != nil {
		t.Fatalf("recovery dispatch err=%v", err)
	}
	var recoveredScheduled int
	if err := database.pool.QueryRow(ctx, `
		SELECT count(*) FROM issues
		WHERE newsletter_id = $1 AND trigger = 'scheduled'
		  AND created_at >= $2
	`, created.Newsletter.ID, recoveryDue.Add(-time.Hour)).Scan(&recoveredScheduled); err != nil {
		t.Fatal(err)
	}
	if recoveredScheduled != 1 {
		t.Fatalf("recovered newsletter dispatched %d scheduled issues", recoveredScheduled)
	}
	record, err = database.GetNewsletter(ctx, account.ID, created.Newsletter.ID)
	if err != nil {
		t.Fatal(err)
	}
	if record.EffectiveRhythmMode != domain.RhythmDaily || record.RhythmThrottledAt != nil {
		t.Fatalf("recovered rhythm=%#v", record.Newsletter)
	}
}

func TestNovelIssueEvidenceComparisonIntegration(t *testing.T) {
	database := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	now := time.Now().UTC()
	account, err := database.SyncAccountIdentity(
		ctx,
		"clerk-evidence-rhythm-"+uuid.NewString(),
		"evidence-rhythm@example.com",
		domain.AccountActive,
		now.UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	requirePaidIntegrationPlan(t, ctx, database, account.ID, now)
	created, err := database.CreateNewsletter(
		ctx,
		account.ID,
		integrationNewsletterInput([]domain.SourceDefinition{{
			Name: "Evidence", URL: "https://example.com/evidence", Limit: 5,
		}}),
		10,
		20,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.pool.Exec(ctx, `
		UPDATE issues SET
			status = 'generated', generation_id = gen_random_uuid(),
			artifact_key = 'test/evidence/previous', completed_at = $2
		WHERE id = $1
	`, created.FirstIssue.ID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	currentIssueID := uuid.NewString()
	if _, err := database.pool.Exec(ctx, `
		INSERT INTO issues (
			id, newsletter_id, trigger, status, available_at,
			public_id, publication_state, created_at
		)
		VALUES ($1, $2, 'manual', 'queued', $3, gen_random_uuid(), 'private', $3)
	`, currentIssueID, created.Newsletter.ID, now); err != nil {
		t.Fatal(err)
	}
	var sourceSpecID string
	if err := database.pool.QueryRow(ctx, `
		SELECT id::text FROM source_specs WHERE newsletter_id = $1 LIMIT 1
	`, created.Newsletter.ID).Scan(&sourceSpecID); err != nil {
		t.Fatal(err)
	}
	endpoint, err := database.UpsertSourceEndpoint(ctx, domain.SourceEndpoint{
		ID: uuid.NewString(), SourceSpecID: sourceSpecID,
		EndpointURL:  "https://example.com/evidence",
		CanonicalURL: "https://example.com/evidence",
		Kind:         domain.SourceKindHTML, Health: "healthy",
		CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	firstSnapshotID, err := database.InsertSourceSnapshot(ctx, domain.SourceSnapshot{
		ID: uuid.NewString(), SourceEndpointID: endpoint.ID,
		ItemKey: "first", Title: "First", CanonicalURL: "https://example.com/first",
		Content: "first evidence", ContentSource: "article", ContentSHA256: "first-sha",
		Metadata: `{}`, FetchedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	secondSnapshotID, err := database.InsertSourceSnapshot(ctx, domain.SourceSnapshot{
		ID: uuid.NewString(), SourceEndpointID: endpoint.ID,
		ItemKey: "second", Title: "Second", CanonicalURL: "https://example.com/second",
		Content: "second evidence", ContentSource: "article", ContentSHA256: "second-sha",
		Metadata: `{}`, FetchedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	for issueID, links := range map[string][]domain.IssueSource{
		created.FirstIssue.ID: {{
			IssueID: created.FirstIssue.ID, SourceSnapshotID: firstSnapshotID,
			Position: 0, CreatedAt: now,
		}},
		currentIssueID: {{
			IssueID: currentIssueID, SourceSnapshotID: firstSnapshotID,
			Position: 0, CreatedAt: now,
		}},
	} {
		if inserted, err := database.InsertIssueSources(ctx, issueID, links); err != nil || !inserted {
			t.Fatalf("freeze issue=%s inserted=%v err=%v", issueID, inserted, err)
		}
	}
	novel, err := database.HasNovelIssueEvidence(ctx, created.Newsletter.ID, currentIssueID)
	if err != nil || novel {
		t.Fatalf("same evidence novel=%v err=%v", novel, err)
	}
	if _, err := database.pool.Exec(ctx, `
		INSERT INTO issue_sources (issue_id, source_snapshot_id, position, created_at)
		VALUES ($1, $2, 1, $3)
	`, currentIssueID, secondSnapshotID, now); err != nil {
		t.Fatal(err)
	}
	novel, err = database.HasNovelIssueEvidence(ctx, created.Newsletter.ID, currentIssueID)
	if err != nil || !novel {
		t.Fatalf("changed evidence novel=%v err=%v", novel, err)
	}
}

func TestReentryBacklogResetPreservesLibraryAndRestoresRhythmIntegration(t *testing.T) {
	database := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	account, err := database.SyncAccountIdentity(
		ctx,
		"clerk-reentry-reset-"+uuid.NewString(),
		"reentry-reset@example.com",
		domain.AccountActive,
		now.UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	requirePaidIntegrationPlan(t, ctx, database, account.ID, now)
	created, err := database.CreateNewsletter(
		ctx,
		account.ID,
		integrationNewsletterInput([]domain.SourceDefinition{{
			Name: "Reset evidence", URL: "https://example.com/reset", Limit: 5,
		}}),
		10,
		20,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.pool.Exec(ctx, `
		UPDATE issues SET status = 'cancelled', completed_at = $2 WHERE id = $1
	`, created.FirstIssue.ID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := database.pool.Exec(ctx, `
		INSERT INTO issues (
			id, newsletter_id, trigger, status, available_at, public_id,
			publication_state, generation_id, artifact_key, dossier_title,
			created_at, completed_at
		)
		SELECT gen_random_uuid(), $2::uuid, 'manual', 'generated', $1::timestamptz,
		       gen_random_uuid(), 'private', gen_random_uuid(),
		       'test/reentry/' || value::text,
		       'Backlog lesson ' || value::text,
		       $1::timestamptz + (value || ' minutes')::interval,
		       $1::timestamptz + (value || ' minutes')::interval
		FROM generate_series(1, 4) AS value
	`, now, created.Newsletter.ID); err != nil {
		t.Fatal(err)
	}
	record, err := database.SetNewsletterRhythm(
		ctx,
		account.ID,
		created.Newsletter.ID,
		RhythmInput{
			Mode: domain.RhythmDaily, SelectedWeekdays: []int{1, 2, 3, 4, 5},
			AutoThrottleEnabled: true, UnopenedLessonLimit: 3,
		},
		now.Add(10*time.Minute),
	)
	if err != nil || record.EffectiveRhythmMode != domain.RhythmWeeklySynthesis {
		t.Fatalf("pre-reset rhythm=%#v err=%v", record.Newsletter, err)
	}
	result, err := database.ResetNewsletterBacklog(
		ctx,
		account.ID,
		created.Newsletter.ID,
		now.Add(11*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.DismissedCount != 3 ||
		result.Newsletter.EffectiveRhythmMode != domain.RhythmDaily ||
		result.Newsletter.RhythmThrottledAt != nil {
		t.Fatalf("reset result=%#v", result)
	}
	candidates, err := database.todayLessonCandidates(ctx, account.ID)
	if err != nil || len(candidates) != 1 || candidates[0].Title != "Backlog lesson 4" {
		t.Fatalf("Today candidates=%#v err=%v", candidates, err)
	}
	library, _, err := database.ListLibraryLessonsPage(
		ctx, account.ID, "", LibraryUnread, 24, nil,
	)
	if err != nil || len(library) != 4 {
		t.Fatalf("dismissed lessons were not preserved in Library: lessons=%#v err=%v", library, err)
	}
	other, err := database.SyncAccountIdentity(
		ctx,
		"clerk-reentry-reset-other-"+uuid.NewString(),
		"reentry-reset-other@example.com",
		domain.AccountActive,
		now.UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.ResetNewsletterBacklog(
		ctx, other.ID, created.Newsletter.ID, now.Add(12*time.Minute),
	); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-account backlog reset err=%v", err)
	}
}

func TestPublicationStatesDefaultPrivateAndRequireFirstPublishReviewIntegration(t *testing.T) {
	database := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	now := time.Now().UTC()
	account, err := database.SyncAccountIdentity(
		ctx,
		"clerk-publication-"+uuid.NewString(),
		"publication@example.com",
		domain.AccountActive,
		now.UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	requirePaidIntegrationPlan(t, ctx, database, account.ID, now)
	site, err := database.ClaimSite(
		ctx, account.ID, "publisher-"+uuid.NewString()[:8], "Publisher",
	)
	if err != nil {
		t.Fatal(err)
	}
	site, err = database.UpdateSite(ctx, account.ID, domain.SitePublic, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	input := integrationNewsletterInput([]domain.SourceDefinition{{
		Name: "Publication source", URL: "https://example.com/publication", Limit: 5,
	}})
	input.SiteVisible = true
	created, err := database.CreateNewsletter(ctx, account.ID, input, 10, 20)
	if err != nil {
		t.Fatal(err)
	}
	if created.Newsletter.LessonPublicationDefault != domain.PublicationDraft ||
		created.FirstIssue.PublicationState != domain.PublicationDraft {
		t.Fatalf("unsafe creation defaults newsletter=%#v issue=%#v", created.Newsletter, created.FirstIssue)
	}
	if _, err := database.pool.Exec(ctx, `
		UPDATE issues SET
			status = 'generated', dossier_title = 'Draft lesson',
			generation_id = gen_random_uuid(), artifact_key = 'test/draft/artifact',
			public_slug = 'draft-lesson', completed_at = $2
		WHERE id = $1
	`, created.FirstIssue.ID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := database.GetPublicIssue(ctx, site.Username, created.FirstIssue.PublicID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("draft leaked through public route: %v", err)
	}
	if _, err := database.SetIssuePublication(
		ctx, account.ID, created.FirstIssue.ID,
		PublicationChange{State: domain.PublicationPublished, Now: now.Add(time.Minute)},
	); err == nil {
		t.Fatal("first publish succeeded without audience confirmation")
	}
	published, err := database.SetIssuePublication(
		ctx, account.ID, created.FirstIssue.ID,
		PublicationChange{
			State: domain.PublicationPublished, AudienceConfirmed: true,
			Now: now.Add(2 * time.Minute),
		},
	)
	if err != nil || published.FirstPublishReviewedAt == nil || published.PublishedAt == nil {
		t.Fatalf("published=%#v err=%v", published, err)
	}
	if _, err := database.GetPublicIssue(ctx, site.Username, created.FirstIssue.PublicID); err != nil {
		t.Fatalf("confirmed lesson is not public: %v", err)
	}
	fingerprint := strings.Repeat("a", 64)
	viewedAt := now.Add(2*time.Minute + time.Second)
	for range 2 {
		if err := database.RecordPublicGrowthEvent(
			ctx, site.Username, created.FirstIssue.PublicID,
			PublicGrowthView, "", fingerprint, viewedAt,
		); err != nil {
			t.Fatal(err)
		}
	}
	if err := database.RecordPublicGrowthEvent(
		ctx, site.Username, created.FirstIssue.PublicID,
		PublicGrowthShare, "linkedin", fingerprint, viewedAt,
	); err != nil {
		t.Fatal(err)
	}
	if err := database.RecordPublicGrowthEvent(
		ctx, site.Username, created.FirstIssue.PublicID,
		PublicGrowthCTAClick, "", fingerprint, viewedAt,
	); err != nil {
		t.Fatal(err)
	}
	converted, err := database.SyncAccountIdentity(
		ctx,
		"clerk-attributed-"+uuid.NewString(),
		"attributed@example.com",
		domain.AccountActive,
		viewedAt.Add(time.Second).UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.pool.Exec(ctx, `
		UPDATE accounts SET created_at = $2, updated_at = $2 WHERE id = $1
	`, converted.ID, viewedAt.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := database.RecordPublicSignupConversion(
		ctx, converted.ID, fingerprint, viewedAt.Add(2*time.Second),
	); err != nil {
		t.Fatal(err)
	}
	analytics, err := database.GetPublicGrowthAnalytics(
		ctx, account.ID, 30, viewedAt.Add(3*time.Second),
	)
	if err != nil || analytics.Views != 1 || analytics.UniqueViewers != 1 ||
		analytics.Shares != 1 || analytics.CTAClicks != 1 ||
		analytics.AttributedSignups != 1 || analytics.AttributedActivations != 0 ||
		analytics.PublishedDossiers != 1 {
		t.Fatalf("public growth analytics=%#v err=%v", analytics, err)
	}
	if err := database.RecordProductEvent(
		ctx, converted.ID, ProductEventActivationCompleted,
		"review", "first-cycle", viewedAt.Add(2500*time.Millisecond),
	); err != nil {
		t.Fatal(err)
	}
	analytics, err = database.GetPublicGrowthAnalytics(
		ctx, account.ID, 30, viewedAt.Add(3*time.Second),
	)
	if err != nil || analytics.AttributedActivations != 1 {
		t.Fatalf("public activation attribution=%#v err=%v", analytics, err)
	}
	otherAnalytics, err := database.GetPublicGrowthAnalytics(
		ctx, converted.ID, 30, viewedAt.Add(3*time.Second),
	)
	if err != nil || otherAnalytics.Views != 0 || otherAnalytics.AttributedSignups != 0 {
		t.Fatalf("cross-owner public analytics=%#v err=%v", otherAnalytics, err)
	}
	if err := database.RequestPublicPathFollow(
		ctx, site.Username, created.FirstIssue.PublicID,
		"reader@example.com", viewedAt.Add(4*time.Second),
	); err != nil {
		t.Fatal(err)
	}
	followClaim, err := database.ClaimNextPublicFollowDelivery(
		ctx, viewedAt.Add(5*time.Second), time.Minute, 3,
	)
	if err != nil || followClaim == nil || followClaim.Kind != "confirmation" ||
		followClaim.Email != "reader@example.com" || followClaim.Token == "" {
		t.Fatalf("public follow confirmation claim=%#v err=%v", followClaim, err)
	}
	if err := database.CompletePublicFollowDelivery(
		ctx, followClaim.ID, followClaim.ClaimToken, "resend-follow-confirmation",
		viewedAt.Add(6*time.Second),
	); err != nil {
		t.Fatal(err)
	}
	if err := database.ConfirmPublicPathFollow(
		ctx, followClaim.Token, viewedAt.Add(7*time.Second),
	); err != nil {
		t.Fatal(err)
	}
	analytics, err = database.GetPublicGrowthAnalytics(
		ctx, account.ID, 30, viewedAt.Add(8*time.Second),
	)
	if err != nil || analytics.Follows != 1 {
		t.Fatalf("confirmed follow analytics=%#v err=%v", analytics, err)
	}
	nextIssue := uuid.NewString()
	nextPublicID := uuid.NewString()
	if _, err := database.pool.Exec(ctx, `
		INSERT INTO issues (
		  id, newsletter_id, trigger, status, available_at, public_id,
		  public_slug, dossier_title, generation_id, artifact_key,
		  publication_state, published_at, completed_at, created_at
		)
		VALUES ($1, $2, 'manual', 'generated', $3, $4, 'next-public-lesson',
		        'Next public lesson', gen_random_uuid(), 'test/next-public-artifact',
		        'published', $3, $3, $3)
	`, nextIssue, created.Newsletter.ID, viewedAt.Add(9*time.Second), nextPublicID); err != nil {
		t.Fatal(err)
	}
	dispatched, err := database.DispatchPublicFollowUpdates(
		ctx, viewedAt.Add(10*time.Second), 10,
	)
	if err != nil || dispatched != 1 {
		t.Fatalf("public follow updates dispatched=%d err=%v", dispatched, err)
	}
	updateClaim, err := database.ClaimNextPublicFollowDelivery(
		ctx, viewedAt.Add(11*time.Second), time.Minute, 3,
	)
	if err != nil || updateClaim == nil || updateClaim.Kind != "update" ||
		updateClaim.IssueTitle != "Next public lesson" || updateClaim.Token == "" {
		t.Fatalf("public follow update claim=%#v err=%v", updateClaim, err)
	}
	if err := database.CompletePublicFollowDelivery(
		ctx, updateClaim.ID, updateClaim.ClaimToken, "resend-follow-update",
		viewedAt.Add(12*time.Second),
	); err != nil {
		t.Fatal(err)
	}
	if err := database.UnsubscribePublicPathFollow(
		ctx, updateClaim.Token, viewedAt.Add(13*time.Second),
	); err != nil {
		t.Fatal(err)
	}
	dispatched, err = database.DispatchPublicFollowUpdates(
		ctx, viewedAt.Add(14*time.Second), 10,
	)
	if err != nil || dispatched != 0 {
		t.Fatalf("unsubscribed follower received updates=%d err=%v", dispatched, err)
	}
	unpublished, err := database.SetIssuePublication(
		ctx, account.ID, created.FirstIssue.ID,
		PublicationChange{State: domain.PublicationDraft, Now: now.Add(3 * time.Minute)},
	)
	if err != nil || unpublished.PublishedAt != nil || unpublished.FirstPublishReviewedAt == nil {
		t.Fatalf("unpublished=%#v err=%v", unpublished, err)
	}
	if _, err := database.GetPublicIssue(ctx, site.Username, created.FirstIssue.PublicID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unpublished lesson remained public: %v", err)
	}
	if err := database.SetNewsletterPublicationDefault(
		ctx, account.ID, created.Newsletter.ID,
		domain.PublicationPublished, false, now.Add(4*time.Minute),
	); err == nil {
		t.Fatal("auto-publish default accepted without confirmation")
	}
	if err := database.SetNewsletterPublicationDefault(
		ctx, account.ID, created.Newsletter.ID,
		domain.PublicationPublished, true, now.Add(5*time.Minute),
	); err != nil {
		t.Fatal(err)
	}
	manual, err := database.EnqueueManualIssue(ctx, account.ID, created.Newsletter.ID, 20)
	if err != nil || manual.PublicationState != domain.PublicationPublished {
		t.Fatalf("confirmed auto-publish issue=%#v err=%v", manual, err)
	}
	other, err := database.SyncAccountIdentity(
		ctx,
		"clerk-publication-other-"+uuid.NewString(),
		"publication-other@example.com",
		domain.AccountActive,
		now.UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.SetIssuePublication(
		ctx, other.ID, created.FirstIssue.ID,
		PublicationChange{State: domain.PublicationPrivate, Now: now.Add(6 * time.Minute)},
	); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-account publication change err=%v", err)
	}
}

func TestOnboardingDraftResumeAndCompletionIntegration(t *testing.T) {
	database := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	now := time.Now().UTC()
	account, err := database.SyncAccountIdentity(
		ctx,
		"clerk-onboarding-draft-"+uuid.NewString(),
		"onboarding-draft@example.com",
		domain.AccountActive,
		now.UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	requirePaidIntegrationPlan(t, ctx, database, account.ID, now)
	first, err := database.SaveOnboardingDraft(
		ctx,
		account.ID,
		uuid.NewString(),
		0,
		1,
		OnboardingDraftPayload{
			Topic:        "AI evaluation",
			LearnerLevel: "intermediate",
			SourceMode:   domain.SourceModeDiscovered,
			Active:       true,
		},
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	resumed, err := database.SaveOnboardingDraft(
		ctx,
		account.ID,
		first.ID,
		first.Revision,
		3,
		OnboardingDraftPayload{
			Topic:            "AI evaluation",
			LearnerLevel:     "intermediate",
			LearnerGoal:      "assess product claims",
			LessonMinutes:    20,
			ScheduleTime:     "08:00",
			TimeZone:         "UTC",
			SourceMode:       domain.SourceModeDiscovered,
			SourceReviewMode: domain.SourceReviewBeforeLesson,
			Active:           true,
		},
		now.Add(time.Second),
	)
	if err != nil || resumed.ID != first.ID || resumed.Step != 3 {
		t.Fatalf("resumed=%#v first=%#v err=%v", resumed, first, err)
	}
	if _, err := database.SaveOnboardingDraft(
		ctx,
		account.ID,
		first.ID,
		first.Revision,
		2,
		first.Payload,
		now.Add(1500*time.Millisecond),
	); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale onboarding revision err=%v", err)
	}
	loaded, err := database.GetOnboardingDraft(ctx, account.ID)
	if err != nil || loaded == nil || loaded.ID != first.ID ||
		loaded.Payload.LearnerGoal != "assess product claims" {
		t.Fatalf("loaded=%#v err=%v", loaded, err)
	}
	if otherDraft, err := database.GetOnboardingDraft(ctx, uuid.NewString()); err != nil || otherDraft != nil {
		t.Fatalf("cross-account draft=%#v err=%v", otherDraft, err)
	}
	if err := database.RecordOnboardingPreviewReached(
		ctx, account.ID, first.ID, now.Add(2*time.Second),
	); err != nil {
		t.Fatal(err)
	}
	var progressEvents int
	if err := database.pool.QueryRow(ctx, `
		SELECT count(*) FROM product_events
		WHERE account_id = $1 AND subject_type = 'onboarding'
		  AND event_name IN (
		    'onboarding_started', 'onboarding_intent_completed',
		    'onboarding_sources_completed', 'source_policy_selected',
		    'source_preview_reached'
		  )
	`, account.ID).Scan(&progressEvents); err != nil {
		t.Fatal(err)
	}
	if progressEvents != 5 {
		t.Fatalf("progress events=%d, want 5", progressEvents)
	}
	input := integrationNewsletterInput(nil)
	input.SourceMode = domain.SourceModeDiscovered
	input.SourceReviewMode = domain.SourceReviewBeforeLesson
	input.OnboardingDraftID = first.ID
	input.OnboardingDraftRevision = first.Revision
	if _, err := database.CreateNewsletter(ctx, account.ID, input, 10, 5); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale onboarding completion err=%v", err)
	}
	var newsletterCount int
	if err := database.pool.QueryRow(ctx, `
		SELECT count(*) FROM newsletters WHERE owner_account_id = $1
	`, account.ID).Scan(&newsletterCount); err != nil || newsletterCount != 0 {
		t.Fatalf("stale completion newsletters=%d err=%v", newsletterCount, err)
	}
	input.OnboardingDraftRevision = resumed.Revision
	if _, err := database.CreateNewsletter(ctx, account.ID, input, 10, 5); err != nil {
		t.Fatal(err)
	}
	loaded, err = database.GetOnboardingDraft(ctx, account.ID)
	if err != nil || loaded != nil {
		t.Fatalf("completed draft=%#v err=%v", loaded, err)
	}
	if _, err := database.SaveOnboardingDraft(
		ctx,
		account.ID,
		first.ID,
		resumed.Revision,
		3,
		resumed.Payload,
		now.Add(2500*time.Millisecond),
	); !errors.Is(err, ErrConflict) {
		t.Fatalf("completed onboarding draft was resurrected: %v", err)
	}
	var confirmed int
	if err := database.pool.QueryRow(ctx, `
		SELECT count(*) FROM product_events
		WHERE account_id = $1 AND event_name = 'onboarding_confirmed'
		  AND subject_id = $2
	`, account.ID, first.ID).Scan(&confirmed); err != nil || confirmed != 1 {
		t.Fatalf("confirmed=%d err=%v", confirmed, err)
	}
	abandoned, err := database.SaveOnboardingDraft(
		ctx,
		account.ID,
		uuid.NewString(),
		0,
		1,
		OnboardingDraftPayload{
			Topic:      "A different question",
			SourceMode: domain.SourceModeDiscovered,
		},
		now.Add(3*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.DeleteOnboardingDraft(
		ctx,
		account.ID,
		abandoned.ID,
		abandoned.Revision+1,
		true,
		now.Add(3500*time.Millisecond),
	); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale onboarding discard err=%v", err)
	}
	if err := database.DeleteOnboardingDraft(
		ctx,
		account.ID,
		abandoned.ID,
		abandoned.Revision,
		true,
		now.Add(4*time.Second),
	); err != nil {
		t.Fatal(err)
	}
	if err := database.pool.QueryRow(ctx, `
		SELECT count(*) FROM product_events
		WHERE account_id = $1 AND event_name = 'onboarding_abandoned'
		  AND subject_id = $2
	`, account.ID, abandoned.ID).Scan(&confirmed); err != nil || confirmed != 1 {
		t.Fatalf("abandoned=%d err=%v", confirmed, err)
	}
}

func TestOperatorPublicHoldIsExactAuditedAndIdempotentIntegration(t *testing.T) {
	database := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	now := time.Now().UTC()
	owner, err := database.SyncAccountIdentity(
		ctx, "clerk-rights-owner-"+uuid.NewString(), "rights-owner@example.com",
		domain.AccountActive, now.UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	requirePaidIntegrationPlan(t, ctx, database, owner.ID, now)
	operator, err := database.SyncAccountIdentity(
		ctx, "clerk-rights-operator-"+uuid.NewString(), "rights-operator@example.com",
		domain.AccountActive, now.Add(time.Millisecond).UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	username := "rights-" + uuid.NewString()[:8]
	if _, err := database.ClaimSite(ctx, owner.ID, username, "Rights Test"); err != nil {
		t.Fatal(err)
	}
	created, err := database.CreateNewsletter(
		ctx, owner.ID, integrationNewsletterInput([]domain.SourceDefinition{{
			Name: "Public source", URL: "https://example.com/article", Limit: 5,
		}}), 10, 5,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.pool.Exec(ctx, `
		UPDATE personal_sites SET visibility = 'public' WHERE owner_account_id = $1
	`, owner.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.pool.Exec(ctx, `
		UPDATE newsletters SET site_visible = true WHERE id = $1
	`, created.Newsletter.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.pool.Exec(ctx, `
		UPDATE issues SET status = 'generated', publication_state = 'published',
		  public_slug = 'rights-test', dossier_title = 'Rights test',
		  generation_id = gen_random_uuid(), artifact_key = 'test/rights-artifact',
		  published_at = $1, completed_at = $1
		WHERE id = $2
	`, now, created.FirstIssue.ID); err != nil {
		t.Fatal(err)
	}

	result, err := database.HoldPublicIssueByOperator(
		ctx, operator.ID, username, created.FirstIssue.PublicID,
		"RIGHTS-1042", "Reviewing a verified rights-holder removal request.",
		now.Add(time.Second),
	)
	if err != nil || result.AlreadyHeld {
		t.Fatalf("first operator hold result=%#v err=%v", result, err)
	}
	if _, err := database.GetPublicIssue(
		ctx, username, created.FirstIssue.PublicID,
	); !errors.Is(err, ErrNotFound) {
		t.Fatalf("held Dossier remained public: %v", err)
	}
	result, err = database.HoldPublicIssueByOperator(
		ctx, operator.ID, username, created.FirstIssue.PublicID,
		"RIGHTS-1042", "Reviewing a verified rights-holder removal request.",
		now.Add(2*time.Second),
	)
	if err != nil || !result.AlreadyHeld {
		t.Fatalf("repeat operator hold result=%#v err=%v", result, err)
	}
	var actionCount int
	var actorAccountID, actionReason string
	if err := database.pool.QueryRow(ctx, `
		SELECT count(*), max(actor_account_id::text), max(reason)
		FROM public_moderation_actions
		WHERE issue_id = $1 AND action = 'publication_held'
		  AND reason LIKE 'Rights-holder review pending%'
	`, created.FirstIssue.ID).Scan(&actionCount, &actorAccountID, &actionReason); err != nil {
		t.Fatal(err)
	}
	if actionCount != 1 || actorAccountID != operator.ID ||
		!strings.Contains(actionReason, "RIGHTS-1042") {
		t.Fatalf("operator audit count=%d actor=%q reason=%q", actionCount, actorAccountID, actionReason)
	}
	if _, err := database.HoldPublicIssueByOperator(
		ctx, owner.ID, username, created.FirstIssue.PublicID,
		"RIGHTS-1043", "Owner must use the normal publishing moderation path.",
		now.Add(3*time.Second),
	); !errors.Is(err, ErrForbidden) {
		t.Fatalf("owner accepted as independent operator: %v", err)
	}
	if _, err := database.HoldPublicIssueByOperator(
		ctx, operator.ID, username, uuid.NewString(),
		"RIGHTS-1044", "This exact public identifier does not exist.",
		now.Add(4*time.Second),
	); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown public Dossier hold err=%v", err)
	}
}

func TestSourceRetrievalPolicyIsAppendOnlyReversibleAndDomainAwareIntegration(t *testing.T) {
	database := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	now := time.Now().UTC()
	operator, err := database.SyncAccountIdentity(
		ctx, "clerk-source-policy-"+uuid.NewString(), "source-policy@example.com",
		domain.AccountActive, now.UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.RecordSourceRetrievalPolicy(
		ctx, operator.ID, SourcePolicyRegistrableDomain, "Example.COM.",
		SourcePolicyBlock, "RIGHTS-2042", "Verified domain-wide retrieval request.", now,
	); err != nil {
		t.Fatal(err)
	}
	for _, rawURL := range []string{
		"https://example.com/article", "https://docs.example.com/guide",
	} {
		allowed, err := database.SourceURLAllowed(ctx, rawURL)
		if err != nil || allowed {
			t.Fatalf("blocked domain URL %q allowed=%t err=%v", rawURL, allowed, err)
		}
	}
	allowed, err := database.SourceURLAllowed(ctx, "https://example.org/article")
	if err != nil || !allowed {
		t.Fatalf("unrelated URL allowed=%t err=%v", allowed, err)
	}
	if err := database.RecordSourceRetrievalPolicy(
		ctx, operator.ID, SourcePolicyRegistrableDomain, "example.com",
		SourcePolicyBlock, "RIGHTS-2042", "Duplicate domain-wide retrieval request.", now.Add(time.Second),
	); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate block err=%v", err)
	}
	if err := database.RecordSourceRetrievalPolicy(
		ctx, operator.ID, SourcePolicyRegistrableDomain, "example.com",
		SourcePolicyUnblock, "RIGHTS-2042", "Authorized resolution restores retrieval.", now.Add(2*time.Second),
	); err != nil {
		t.Fatal(err)
	}
	allowed, err = database.SourceURLAllowed(ctx, "https://docs.example.com/guide")
	if err != nil || !allowed {
		t.Fatalf("restored URL allowed=%t err=%v", allowed, err)
	}
	var events int
	var currentAction string
	if err := database.pool.QueryRow(ctx, `
		SELECT count(*) FROM source_retrieval_policy_events
		WHERE scope = 'registrable_domain' AND value = 'example.com'
	`).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if err := database.pool.QueryRow(ctx, `
		SELECT action FROM current_source_retrieval_policy
		WHERE scope = 'registrable_domain' AND value = 'example.com'
	`).Scan(&currentAction); err != nil {
		t.Fatal(err)
	}
	if events != 2 || currentAction != SourcePolicyUnblock {
		t.Fatalf("events=%d current action=%q", events, currentAction)
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
	requirePaidIntegrationPlan(t, ctx, database, account.ID, time.Now().UTC())
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
		Role: domain.SourceRoleOfficialPrimary, RankingVersion: "source-rank-v2",
		ScoreComponents: domain.SourceScoreComponents{
			Relevance: 20, Authority: 14, Primaryness: 8,
		},
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
		catalog[0].Health != "healthy" ||
		catalog[0].Role != domain.SourceRoleOfficialPrimary ||
		catalog[0].RankingVersion != "source-rank-v2" {
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

func requirePaidIntegrationPlan(
	t *testing.T,
	ctx context.Context,
	database *Store,
	accountID string,
	now time.Time,
) {
	t.Helper()
	if err := activateIntegrationPlan(ctx, database, accountID, "pro", now); err != nil {
		t.Fatal(err)
	}
}

func integrationNewsletterInput(sources []domain.SourceDefinition) NewsletterInput {
	return NewsletterInput{
		Name: "Source integration", Topic: "source intelligence",
		LearnerLevel: "intermediate", LearnerGoal: "build a practical understanding",
		LessonMinutes: 20, SourceMode: domain.SourceModeProvided, Sources: sources,
		ScheduleHour: 8, TimeZone: "UTC", Active: true,
	}
}
