package artifact

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/VatsalP117/learnloom/internal/domain"
)

func TestArtifactKeyValidation(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"", ".", "..", "../escape", "bad/part", `bad\part`} {
		if safePart(value) {
			t.Errorf("%q should be rejected as a part", value)
		}
	}
	for _, value := range []string{"accounts/a/issues/i/g.json.gz", "a-b_c/123.json.gz"} {
		if !safeKey(value) {
			t.Errorf("%q should be accepted as a key", value)
		}
	}
	for _, value := range []string{"/absolute", "a/../b", "a//b"} {
		if safeKey(value) {
			t.Errorf("%q should be rejected as a key", value)
		}
	}
}

func TestArtifactKeyForMatchesPutNamespace(t *testing.T) {
	t.Parallel()
	key, err := KeyFor("account", "stream", "issue", "generation")
	if err != nil {
		t.Fatal(err)
	}
	if key != "accounts/account/newsletters/stream/issues/issue/generation.json.gz" {
		t.Fatalf("key=%q", key)
	}
	if _, err := KeyFor("../account", "stream", "issue", "generation"); err == nil {
		t.Fatal("unsafe artifact namespace was accepted")
	}
}

func TestCompressedCanonicalArtifactRoundTrip(t *testing.T) {
	t.Parallel()
	dossierValue := domain.Dossier{
		Version: 2,
		Date:    "2026-07-27",
		Title:   "Compressed canonical storage",
		Lesson:  strings.Repeat("A grounded lesson paragraph. ", 200),
	}
	canonical, err := json.Marshal(storedArtifact{
		FormatVersion: artifactFormatVersion,
		RenderVersion: artifactRenderVersion,
		Dossier:       dossierValue,
	})
	if err != nil {
		t.Fatal(err)
	}
	compressed, err := compressArtifact(canonical)
	if err != nil {
		t.Fatal(err)
	}
	if len(compressed) >= len(canonical) {
		t.Fatalf("compressed artifact is not smaller: compressed=%d canonical=%d", len(compressed), len(canonical))
	}
	decoded, err := decompressArtifact(compressed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded, canonical) {
		t.Fatal("decompressed artifact differs from canonical JSON")
	}
	var stored storedArtifact
	if err := json.Unmarshal(decoded, &stored); err != nil {
		t.Fatal(err)
	}
	artifactValue := renderArtifact(stored.Dossier)
	if artifactValue.Dossier.Title != dossierValue.Title ||
		!strings.Contains(artifactValue.Markdown, dossierValue.Lesson) ||
		!strings.Contains(artifactValue.HTML, "A grounded lesson paragraph.") {
		t.Fatalf("unexpected rendered artifact: %#v", artifactValue)
	}
}

func TestDecompressArtifactRejectsOversizedContent(t *testing.T) {
	t.Parallel()
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := io.CopyN(writer, zeroReader{}, maximumArtifactBytes+1); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := decompressArtifact(compressed.Bytes()); err == nil ||
		!strings.Contains(err.Error(), "exceeds the size limit") {
		t.Fatalf("expected size-limit error, got %v", err)
	}
}

type zeroReader struct{}

func (zeroReader) Read(buffer []byte) (int, error) {
	clear(buffer)
	return len(buffer), nil
}
