package editorial

import "testing"

func TestStarterReviewManifestSeparatesStructureFromHumanRelease(t *testing.T) {
	manifest := testManifest()
	if err := ValidateStarterReviewManifest(manifest); err != nil {
		t.Fatal(err)
	}
	if err := ValidateStarterReviewRelease(manifest); err == nil {
		t.Fatal("pending human reviews passed the release gate")
	}
}

func TestStarterReviewReleaseRequiresCompleteNamedEvidence(t *testing.T) {
	manifest := testManifest()
	for index := range manifest.Reviews {
		review := &manifest.Reviews[index]
		review.Decision = "approve"
		review.Reviewer = "Qualified reviewer"
		review.ReviewedOn = "2026-08-12"
		review.LessonArtifact = "docs/release-evidence/starter-lessons/lesson.md"
		review.OutcomeUseful = true
		review.SourcesVerified = true
		review.LessonAnswerable = true
		review.ClaimsChecked = true
		review.Notes = "Approved after source and lesson review."
		review.Findings = []ClaimFinding{{
			Claim: "Material claim", SourceURL: "https://example.com/source",
			Disposition: "supported", Note: "Direct support.",
		}}
	}
	if err := ValidateStarterReviewRelease(manifest); err != nil {
		t.Fatal(err)
	}
	manifest.Reviews[0].ClaimsChecked = false
	if err := ValidateStarterReviewRelease(manifest); err == nil {
		t.Fatal("approval without claim checks passed the release gate")
	}
}

func testManifest() StarterReviewManifest {
	manifest := StarterReviewManifest{Version: "test", CatalogVersion: 3}
	for _, id := range []string{"one", "two", "three", "four", "five"} {
		manifest.Reviews = append(manifest.Reviews, StarterReview{
			TemplateID: id, TemplateVersion: 1, Name: id,
			SourceURLs: []string{"https://example.com/source"}, Decision: "pending",
		})
	}
	return manifest
}
