package failure

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestEveryStableFailureCodeHasABehavioralFixture(t *testing.T) {
	t.Parallel()
	fixtures := map[string]string{
		CodeModelContractUnsatisfied:     "dossier/TestStructuredStagePreservesContractFailureClassification",
		CodeModelProviderUnavailable:     "dossier/TestStageExecutionFailureRetriesTransientProviderRequest",
		CodeModelRequestRejected:         "dossier/TestStageExecutionFailureDoesNotRetryPermanentProviderRequest",
		CodeModelOutputTruncated:         "execution/TestTruncatedGenerationIncrementsMetricAndRetries",
		CodeModelOutputEmpty:             "dossier/TestStageExecutionFailureClassifiesEmptyOutputAsContentQuality",
		CodeGenerationInterrupted:        "dossier/TestStageExecutionFailureClassifiesDeadlineAsRetryableInterruption",
		CodeNoNewLearningSignal:          "dossier/TestGeneratorDefersRepeatedLearningBeforeDownstreamStages",
		CodeSourceDiscoveryUnavailable:   "source/TestDiscoveryOutageIsRetryableInfrastructure",
		CodeSourceEvidenceNeedsAttention: "source/TestProvidedInsufficientEvidenceRequiresSourceCorrection",
		CodeNoWorthwhileEvidence:         "source/TestDiscoveredInsufficientEvidenceIsAnHonestDeferral",
		CodeNoNewEvidence:                "execution/TestEvidenceLedRhythmDefersUnchangedEvidence",
		CodeWorkerClaimExpired:           "store/TestRecoverExpiredIssueClaims",
	}
	want := make([]string, 0, len(fixtures))
	for code, fixture := range fixtures {
		if strings.TrimSpace(fixture) == "" {
			t.Fatalf("stable failure code %q has no fixture", code)
		}
		want = append(want, code)
	}
	slices.Sort(want)
	got := append([]string(nil), StableCodes...)
	slices.Sort(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("stable codes=%v fixture codes=%v", got, want)
	}
}

func TestDescribeKeepsInternalDetailOutOfPublicMessage(t *testing.T) {
	secret := errors.New("provider rejected secret-token")
	detail := Describe(New(
		"model_contract_unsatisfied",
		CategoryContentQuality,
		"editor",
		true,
		PublicInternal,
		secret,
	))
	if detail.Code != "model_contract_unsatisfied" ||
		detail.Category != CategoryContentQuality ||
		detail.Stage != "editor" ||
		!detail.Retryable ||
		detail.IncidentID == "" {
		t.Fatalf("unexpected detail: %#v", detail)
	}
	if !strings.Contains(detail.Internal, "secret-token") ||
		strings.Contains(detail.PublicMessage, "secret-token") {
		t.Fatalf("failure detail leaked: %#v", detail)
	}
}
