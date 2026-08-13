package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VatsalP117/learnloom/internal/failure"
)

func TestLaunchEvidenceReportsUseExplicitRealUserClassification(t *testing.T) {
	t.Parallel()
	repositoryRoot := filepath.Clean(filepath.Join("..", ".."))
	tests := []struct {
		path       string
		minimumUse int
	}{
		{path: "scripts/product-baseline.sql", minimumUse: 9},
		{path: "scripts/billing-economics.sql", minimumUse: 6},
		{path: "scripts/design-partner-beta-report.sql", minimumUse: 7},
	}
	for _, test := range tests {
		body, err := os.ReadFile(filepath.Join(repositoryRoot, test.path))
		if err != nil {
			t.Fatalf("read %s: %v", test.path, err)
		}
		report := string(body)
		if !strings.Contains(report, "unclassified") {
			t.Errorf("%s does not expose unclassified evidence", test.path)
		}
		if uses := strings.Count(report, "evidence_class = 'real_user'"); uses < test.minimumUse {
			t.Errorf(
				"%s has %d explicit real-user boundaries, want at least %d",
				test.path,
				uses,
				test.minimumUse,
			)
		}
		for _, forbidden := range []string{"primary_email LIKE", "primary_email NOT LIKE", "clerk_user_id LIKE"} {
			if strings.Contains(report, forbidden) {
				t.Errorf("%s uses unsafe identity heuristic %q", test.path, forbidden)
			}
		}
	}
}

func TestProductBaselineUsesCohortAlignedRetainedLessonCost(t *testing.T) {
	t.Parallel()
	body, err := os.ReadFile(filepath.Join("..", "..", "scripts", "product-baseline.sql"))
	if err != nil {
		t.Fatal(err)
	}
	report := string(body)
	for _, expected := range []string{
		"cost per seven-day retained lesson",
		"event.occurred_at > mature.activated_at",
		"event.occurred_at <= mature.activated_at + interval '7 days'",
		"event.event_name IN ('lesson_completed', 'review_attempted')",
		"count(*) AS retained_lessons",
		"cost_per_retained_lesson_usd",
	} {
		if !strings.Contains(report, expected) {
			t.Errorf("retained lesson economics missing %q", expected)
		}
	}
}

func TestSevenDayReturnHasStrictUpperBound(t *testing.T) {
	t.Parallel()
	body, err := os.ReadFile(filepath.Join("..", "..", "scripts", "product-baseline.sql"))
	if err != nil {
		t.Fatal(err)
	}
	report := string(body)
	for _, expected := range []string{
		"later.occurred_at > mature.activated_at",
		"later.occurred_at <= mature.activated_at + interval '7 days'",
	} {
		if !strings.Contains(report, expected) {
			t.Errorf("seven-day retention window missing %q", expected)
		}
	}
	if strings.Contains(report, "later.occurred_at >= mature.activated_at + interval '7 days'") {
		t.Fatal("seven-day return must not count only activity after the retention window")
	}
}

func TestProductBaselineFlagsEveryUnregisteredFailureCode(t *testing.T) {
	t.Parallel()
	body, err := os.ReadFile(filepath.Join("..", "..", "scripts", "product-baseline.sql"))
	if err != nil {
		t.Fatal(err)
	}
	report := string(body)
	for _, code := range failure.StableCodes {
		if !strings.Contains(report, "('"+code+"')") {
			t.Errorf("baseline registered failure list is missing %q", code)
		}
	}
	for _, expected := range []string{
		"unregistered failure-code gate",
		"registered.code IS NULL",
		"any row above requires a safe fixture",
	} {
		if !strings.Contains(report, expected) {
			t.Errorf("baseline unregistered-code gate missing %q", expected)
		}
	}
}

func TestEvidenceClassificationScriptIsAppendOnlyAndAccountIDOnly(t *testing.T) {
	t.Parallel()
	body, err := os.ReadFile(filepath.Join("..", "..", "scripts", "classify-evidence-account.sql"))
	if err != nil {
		t.Fatal(err)
	}
	script := string(body)
	for _, expected := range []string{
		"INSERT INTO account_evidence_classifications",
		"WHERE id = :'account_id'::uuid",
		"active or suspended account not found",
		"ROLLBACK",
	} {
		if !strings.Contains(script, expected) {
			t.Errorf("classification script missing %q", expected)
		}
	}
	for _, forbidden := range []string{"UPDATE account_evidence_classifications", "DELETE FROM account_evidence_classifications", "primary_email", "clerk_user_id"} {
		if strings.Contains(script, forbidden) {
			t.Errorf("classification script contains forbidden operation/data %q", forbidden)
		}
	}
}
