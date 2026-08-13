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
	username := flag.String("username", "", "exact public-site username")
	publicID := flag.String("public-id", "", "exact public Dossier ID")
	confirmPublicID := flag.String("confirm-public-id", "", "repeat the exact public Dossier ID")
	caseReference := flag.String("case-reference", "", "non-PII support or legal case reference")
	reason := flag.String("reason", "", "concise non-PII reason for the hold")
	apply := flag.Bool("apply", false, "apply the hold; omission performs validation only")
	flag.Parse()

	operatorAccountID := strings.TrimSpace(os.Getenv("LEARNLOOM_OPERATOR_ACCOUNT_ID"))
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if err := validateInput(
		*username, *publicID, *confirmPublicID, *caseReference, *reason,
		operatorAccountID, databaseURL, *apply,
	); err != nil {
		fail(err)
	}
	if !*apply {
		fmt.Println("hold input validated; rerun with -apply to change public availability")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	database, err := store.Open(ctx, store.Config{
		URL:              databaseURL,
		MaxConnections:   1,
		MinConnections:   0,
		StatementTimeout: 15 * time.Second,
	})
	if err != nil {
		fail(err)
	}
	defer database.Close()
	if err := database.Ready(ctx); err != nil {
		fail(err)
	}
	result, err := database.HoldPublicIssueByOperator(
		ctx,
		operatorAccountID,
		*username,
		*publicID,
		*caseReference,
		*reason,
		time.Now().UTC(),
	)
	if err != nil {
		fail(err)
	}
	if result.AlreadyHeld {
		fmt.Println("public Dossier was already held; no duplicate audit action was written")
		return
	}
	fmt.Println("public Dossier held and operator action recorded")
}

func validateInput(
	username, publicID, confirmPublicID, caseReference, reason,
	operatorAccountID, databaseURL string,
	apply bool,
) error {
	for name, value := range map[string]string{
		"-username":                     username,
		"-public-id":                    publicID,
		"-confirm-public-id":            confirmPublicID,
		"-case-reference":               caseReference,
		"-reason":                       reason,
		"LEARNLOOM_OPERATOR_ACCOUNT_ID": operatorAccountID,
		"DATABASE_URL":                  databaseURL,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	if publicID != confirmPublicID {
		return fmt.Errorf("-confirm-public-id must exactly match -public-id")
	}
	if _, err := uuid.Parse(strings.TrimPrefix(publicID, "dossier-")); err != nil {
		return fmt.Errorf("-public-id must be a UUID or dossier-prefixed UUID")
	}
	if _, err := uuid.Parse(operatorAccountID); err != nil {
		return fmt.Errorf("LEARNLOOM_OPERATOR_ACCOUNT_ID must be a UUID")
	}
	if len(caseReference) < 3 || len(caseReference) > 80 ||
		len(reason) < 10 || len(reason) > 800 {
		return fmt.Errorf("case reference or reason is outside its allowed length")
	}
	if strings.ContainsAny(caseReference, "\r\n") || strings.ContainsAny(reason, "\r\n") {
		return fmt.Errorf("case reference and reason must be single-line values")
	}
	if !apply {
		return nil
	}
	return nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
