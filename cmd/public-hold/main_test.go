package main

import (
	"testing"

	"github.com/google/uuid"
)

func TestValidateInputRequiresExactConfirmation(t *testing.T) {
	err := validateInput(
		"alan", uuid.NewString(), uuid.NewString(), "RIGHTS-1042",
		"Reviewing a rights-holder request.", uuid.NewString(), "postgres://database", true,
	)
	if err == nil {
		t.Fatal("mismatched public Dossier confirmation was accepted")
	}
}

func TestValidateInputRejectsMultilineAuditValues(t *testing.T) {
	publicID := uuid.NewString()
	err := validateInput(
		"alan", publicID, publicID, "RIGHTS-1042\nprivate detail",
		"Reviewing a rights-holder request.", uuid.NewString(), "postgres://database", true,
	)
	if err == nil {
		t.Fatal("multiline case reference was accepted")
	}
}
