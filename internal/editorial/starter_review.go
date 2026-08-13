// Package editorial validates the human review evidence for canonical starter paths.
package editorial

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type StarterReviewManifest struct {
	Version        string          `json:"version"`
	CatalogVersion int             `json:"catalogVersion"`
	Reviews        []StarterReview `json:"reviews"`
}

type StarterReview struct {
	TemplateID       string         `json:"templateId"`
	TemplateVersion  int            `json:"templateVersion"`
	Name             string         `json:"name"`
	SourceURLs       []string       `json:"sourceUrls"`
	LessonArtifact   string         `json:"lessonArtifact"`
	Reviewer         string         `json:"reviewer"`
	ReviewedOn       string         `json:"reviewedOn"`
	Decision         string         `json:"decision"`
	OutcomeUseful    bool           `json:"outcomeUseful"`
	SourcesVerified  bool           `json:"sourcesVerified"`
	LessonAnswerable bool           `json:"lessonAnswerable"`
	ClaimsChecked    bool           `json:"claimsChecked"`
	Findings         []ClaimFinding `json:"findings"`
	Notes            string         `json:"notes"`
}

type ClaimFinding struct {
	Claim       string `json:"claim"`
	SourceURL   string `json:"sourceUrl"`
	Disposition string `json:"disposition"`
	Note        string `json:"note"`
}

func ValidateStarterReviewManifest(manifest StarterReviewManifest) error {
	if strings.TrimSpace(manifest.Version) == "" || manifest.CatalogVersion < 1 {
		return errors.New("starter review manifest version and catalogVersion are required")
	}
	if len(manifest.Reviews) < 5 || len(manifest.Reviews) > 8 {
		return fmt.Errorf("starter review manifest has %d paths; want 5 to 8", len(manifest.Reviews))
	}
	seen := make(map[string]struct{}, len(manifest.Reviews))
	for index, review := range manifest.Reviews {
		if strings.TrimSpace(review.TemplateID) == "" || review.TemplateVersion < 1 ||
			strings.TrimSpace(review.Name) == "" {
			return fmt.Errorf("starter review %d has incomplete template identity", index+1)
		}
		if _, exists := seen[review.TemplateID]; exists {
			return fmt.Errorf("starter review template %q is duplicated", review.TemplateID)
		}
		seen[review.TemplateID] = struct{}{}
		if len(review.SourceURLs) == 0 || len(review.SourceURLs) > 6 {
			return fmt.Errorf("starter review %q must contain 1 to 6 sources", review.TemplateID)
		}
		for _, rawURL := range review.SourceURLs {
			if !validHTTPSURL(rawURL) {
				return fmt.Errorf("starter review %q has an invalid source URL", review.TemplateID)
			}
		}
		if review.Decision != "pending" && review.Decision != "approve" &&
			review.Decision != "revise" && review.Decision != "reject" {
			return fmt.Errorf("starter review %q has an invalid decision", review.TemplateID)
		}
		if review.ReviewedOn != "" {
			if _, err := time.Parse("2006-01-02", review.ReviewedOn); err != nil {
				return fmt.Errorf("starter review %q has an invalid reviewedOn date", review.TemplateID)
			}
		}
		for findingIndex, finding := range review.Findings {
			if strings.TrimSpace(finding.Claim) == "" || !validHTTPSURL(finding.SourceURL) ||
				(finding.Disposition != "supported" && finding.Disposition != "revised" &&
					finding.Disposition != "removed") {
				return fmt.Errorf("starter review %q finding %d is incomplete", review.TemplateID, findingIndex+1)
			}
		}
	}
	return nil
}

func ValidateStarterReviewRelease(manifest StarterReviewManifest) error {
	if err := ValidateStarterReviewManifest(manifest); err != nil {
		return err
	}
	approved := 0
	for _, review := range manifest.Reviews {
		if review.Decision != "approve" {
			continue
		}
		approved++
		if strings.TrimSpace(review.Reviewer) == "" || review.ReviewedOn == "" ||
			strings.TrimSpace(review.LessonArtifact) == "" || !review.OutcomeUseful ||
			!review.SourcesVerified || !review.LessonAnswerable || !review.ClaimsChecked ||
			len(review.Findings) == 0 || strings.TrimSpace(review.Notes) == "" {
			return fmt.Errorf("approved starter review %q lacks required human evidence", review.TemplateID)
		}
	}
	if approved < 5 {
		return fmt.Errorf("starter review release has %d approvals; at least 5 are required", approved)
	}
	return nil
}

func validHTTPSURL(raw string) bool {
	parsed, err := url.Parse(raw)
	return err == nil && parsed.Scheme == "https" && parsed.Host != "" &&
		parsed.User == nil && parsed.Fragment == ""
}
