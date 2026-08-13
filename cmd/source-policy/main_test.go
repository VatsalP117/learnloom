package main

import (
	"testing"

	"github.com/VatsalP117/learnloom/internal/store"
	"github.com/google/uuid"
)

func TestValidateInputRequiresExactConfirmation(t *testing.T) {
	err := validateInput(
		store.SourcePolicyExactURL, "https://example.com/a", "https://example.com/b",
		store.SourcePolicyBlock, "RIGHTS-1042", "Verified rights-holder request.",
		uuid.NewString(), "postgres://database",
	)
	if err == nil {
		t.Fatal("mismatched policy confirmation was accepted")
	}
}

func TestValidateInputRejectsUnknownPolicyAction(t *testing.T) {
	err := validateInput(
		store.SourcePolicyRegistrableDomain, "example.com", "example.com", "delete",
		"RIGHTS-1042", "Verified rights-holder request.", uuid.NewString(), "postgres://database",
	)
	if err == nil {
		t.Fatal("unknown policy action was accepted")
	}
}
