package artifact

import "testing"

func TestArtifactKeyValidation(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"", ".", "..", "../escape", "bad/part", `bad\part`} {
		if safePart(value) {
			t.Errorf("%q should be rejected as a part", value)
		}
	}
	for _, value := range []string{"accounts/a/issues/i/g.json", "a-b_c/123.json"} {
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
