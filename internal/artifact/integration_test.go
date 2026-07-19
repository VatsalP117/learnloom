package artifact

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
)

func TestS3ArtifactLifecycleIntegration(t *testing.T) {
	endpoint := os.Getenv("TEST_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("TEST_S3_ENDPOINT is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	storage, err := New(ctx, Config{
		Bucket: os.Getenv("TEST_S3_BUCKET"), Region: "us-east-1",
		Endpoint: endpoint, AccessKeyID: os.Getenv("TEST_S3_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("TEST_S3_SECRET_ACCESS_KEY"), UsePathStyle: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Ready(ctx); err != nil {
		t.Fatal(err)
	}
	saved, err := storage.Put(ctx, PutInput{
		AccountID: "account-test", NewsletterID: "newsletter-test",
		IssueID: "issue-test", GenerationID: "generation-test",
		Artifact: domain.DossierArtifact{
			Dossier:  domain.Dossier{Version: 1, Title: "Integration Dossier"},
			Markdown: "# Integration Dossier",
			HTML:     "<h1>Integration Dossier</h1>",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := storage.Get(ctx, saved.Key)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Dossier.Title != "Integration Dossier" {
		t.Fatalf("unexpected Dossier: %#v", loaded.Dossier)
	}
	if err := storage.DeleteAccount(ctx, "account-test"); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.Get(ctx, saved.Key); err == nil {
		t.Fatal("account artifact still exists after deletion")
	}
}
