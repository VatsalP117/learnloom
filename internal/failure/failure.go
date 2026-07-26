// Package failure owns stable failure classification across Dossier production,
// Issue execution, persistence, and user-facing presentation.
package failure

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type Category string

const (
	CategoryInternal             Category = "internal"
	CategoryInfrastructure       Category = "infrastructure"
	CategoryProvider             Category = "provider"
	CategoryContentQuality       Category = "content_quality"
	CategoryInsufficientEvidence Category = "insufficient_evidence"
	CategoryUserActionable       Category = "user_actionable"
)

const (
	PublicInternal = "We couldn’t prepare this lesson. We’ve been notified, and you can retry now."
	PublicDelayed  = "This lesson is taking longer than expected. We’re retrying automatically."
	PublicSources  = "We couldn’t find enough usable material in the selected sources. Update the sources, then retry."
)

type Detail struct {
	Code          string
	Category      Category
	Stage         string
	Retryable     bool
	PublicMessage string
	IncidentID    string
	Internal      string
}

type Error struct {
	detail Detail
	cause  error
}

func New(
	code string,
	category Category,
	stage string,
	retryable bool,
	publicMessage string,
	cause error,
) error {
	if cause == nil {
		cause = errors.New(code)
	}
	return &Error{
		detail: Detail{
			Code:          normalizeCode(code),
			Category:      category,
			Stage:         strings.TrimSpace(stage),
			Retryable:     retryable,
			PublicMessage: strings.TrimSpace(publicMessage),
			IncidentID:    uuid.NewString(),
		},
		cause: cause,
	}
}

func (e *Error) Error() string {
	if e.detail.Stage == "" {
		return e.cause.Error()
	}
	return fmt.Sprintf("%s stage: %v", e.detail.Stage, e.cause)
}

func (e *Error) Unwrap() error { return e.cause }

func Describe(err error) Detail {
	if err == nil {
		return Detail{
			Code:          "unknown_error",
			Category:      CategoryInternal,
			PublicMessage: PublicInternal,
			IncidentID:    uuid.NewString(),
			Internal:      "unknown error",
		}
	}
	var classified *Error
	if errors.As(err, &classified) {
		detail := classified.detail
		detail.Internal = strings.Join(strings.Fields(err.Error()), " ")
		if detail.PublicMessage == "" {
			detail.PublicMessage = PublicInternal
		}
		if detail.IncidentID == "" {
			detail.IncidentID = uuid.NewString()
		}
		return detail
	}
	return Detail{
		Code:          "internal_error",
		Category:      CategoryInternal,
		Retryable:     true,
		PublicMessage: PublicInternal,
		IncidentID:    uuid.NewString(),
		Internal:      strings.Join(strings.Fields(err.Error()), " "),
	}
}

func normalizeCode(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "internal_error"
	}
	var result strings.Builder
	underscore := false
	for _, character := range value {
		switch {
		case character >= 'a' && character <= 'z', character >= '0' && character <= '9':
			result.WriteRune(character)
			underscore = false
		default:
			if result.Len() > 0 && !underscore {
				result.WriteByte('_')
				underscore = true
			}
		}
	}
	return strings.Trim(result.String(), "_")
}
