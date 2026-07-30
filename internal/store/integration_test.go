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
	claim, err := database.ClaimNextIssue(
		ctx,
		time.Now().UTC(),
		5*time.Minute,
		1,
		5,
		100,
		IssueAttemptContext{WorkerID: "test-worker", DeploymentVersion: "test"},
	)
	if err != nil || claim == nil || claim.Issue.ID != issue.ID {
		t.Fatalf("claim=%#v err=%v", claim, err)
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
		IssueAttemptContext{WorkerID: "test-worker", DeploymentVersion: "test"},
	)
	if err != nil || claim == nil || claim.Issue.ID != issue.ID {
		t.Fatalf("released retry claim=%#v err=%v", claim, err)
	}
	err = database.CompleteIssue(ctx, issue.ID, CompleteIssueInput{
		ClaimToken: claim.Token, GenerationID: uuid.NewString(),
		ArtifactKey: "accounts/a/dossier.json", Checksum: "abc", Bytes: 100,
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
		if err != nil || len(matches) != 1 || matches[0].ID != issue.ID ||
			len(matches[0].Concepts) != 2 || cursor != nil {
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
	reviews, err = database.ListWorkspaceReviews(ctx, account.ID, 8, now.Add(38*time.Second))
	if err != nil || len(reviews) != 3 || reviews[0].IssueID != issue.ID {
		t.Fatalf("activated workspace reviews=%#v err=%v", reviews, err)
	}
	assessmentKey := uuid.NewString()
	assessed, err := database.AssessReview(
		ctx,
		account.ID,
		reviews[0].ID,
		assessmentKey,
		ReviewSolid,
		now.Add(39*time.Second),
	)
	if err != nil || assessed.Stage != 2 ||
		!assessed.DueAt.Equal(now.Add(39*time.Second+7*24*time.Hour)) {
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
		learnerState.RecallConfidence != recall {
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
	if err := database.RecordOwnedLessonEvent(
		ctx,
		account.ID,
		issue.ID,
		ProductEventReviewAttempted,
		now.Add(39*time.Second),
	); err != nil {
		t.Fatalf("record first retrieval: %v", err)
	}
	retention, err := database.GetRetentionState(
		ctx,
		account.ID,
		now.Add(8*24*time.Hour),
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
		now.Add(8*24*time.Hour),
	); err != nil {
		t.Fatalf("record seven-day return: %v", err)
	}
	retention, err = database.GetRetentionState(
		ctx,
		account.ID,
		now.Add(8*24*time.Hour+time.Minute),
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
