package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/VatsalP117/learnloom/internal/store"
	"github.com/google/uuid"
)

func main() {
	scope := flag.String("scope", "", "policy scope: exact_url or registrable_domain")
	value := flag.String("value", "", "exact public URL or registrable domain")
	confirmValue := flag.String("confirm-value", "", "repeat the exact policy value")
	action := flag.String("action", "", "policy action: block or unblock")
	caseReference := flag.String("case-reference", "", "non-PII support or legal case reference")
	reason := flag.String("reason", "", "concise non-PII reason for the policy action")
	apply := flag.Bool("apply", false, "record the policy event; omission performs validation only")
	flag.Parse()

	operatorAccountID := strings.TrimSpace(os.Getenv("LEARNLOOM_OPERATOR_ACCOUNT_ID"))
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if err := validateInput(*scope, *value, *confirmValue, *action, *caseReference,
		*reason, operatorAccountID, databaseURL); err != nil {
		fail(err)
	}
	if !*apply {
		fmt.Println("source-policy input validated; rerun with -apply to record the action")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	database, err := store.Open(ctx, store.Config{
		URL: databaseURL, MaxConnections: 1, MinConnections: 0,
		StatementTimeout: 15 * time.Second,
	})
	if err != nil {
		fail(err)
	}
	defer database.Close()
	if err := database.Ready(ctx); err != nil {
		fail(err)
	}
	if err := database.RecordSourceRetrievalPolicy(
		ctx, operatorAccountID, *scope, *value, *action, *caseReference, *reason,
		time.Now().UTC(),
	); err != nil {
		fail(err)
	}
	fmt.Printf("source retrieval policy action %q recorded for %s\n", *action, *scope)
}

func validateInput(
	scope, value, confirmValue, action, caseReference, reason,
	operatorAccountID, databaseURL string,
) error {
	for name, value := range map[string]string{
		"-scope": scope, "-value": value, "-confirm-value": confirmValue,
		"-action": action, "-case-reference": caseReference, "-reason": reason,
		"LEARNLOOM_OPERATOR_ACCOUNT_ID": operatorAccountID, "DATABASE_URL": databaseURL,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	if value != confirmValue {
		return fmt.Errorf("-confirm-value must exactly match -value")
	}
	if scope != store.SourcePolicyExactURL && scope != store.SourcePolicyRegistrableDomain {
		return fmt.Errorf("-scope must be exact_url or registrable_domain")
	}
	if action != store.SourcePolicyBlock && action != store.SourcePolicyUnblock {
		return fmt.Errorf("-action must be block or unblock")
	}
	if _, err := uuid.Parse(operatorAccountID); err != nil {
		return fmt.Errorf("LEARNLOOM_OPERATOR_ACCOUNT_ID must be a UUID")
	}
	if len(value) > 2048 || len(caseReference) < 3 || len(caseReference) > 80 ||
		len(reason) < 10 || len(reason) > 800 {
		return fmt.Errorf("value, case reference, or reason is outside its allowed length")
	}
	if strings.ContainsAny(value+caseReference+reason, "\r\n") {
		return fmt.Errorf("value, case reference, and reason must be single-line values")
	}
	return nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
