package store

import "testing"

func TestNormalizeFollowerEmailRequiresBareMailbox(t *testing.T) {
	t.Parallel()
	valid, err := normalizeFollowerEmail(" Reader@Example.com ")
	if err != nil || valid != "reader@example.com" {
		t.Fatalf("valid email=%q err=%v", valid, err)
	}
	for _, value := range []string{"", "not-an-email", "Name <reader@example.com>", "a@"} {
		if _, err := normalizeFollowerEmail(value); err == nil {
			t.Errorf("invalid email %q was accepted", value)
		}
	}
}

func TestRandomPublicTokenIsStrongAndUnique(t *testing.T) {
	t.Parallel()
	first, err := randomPublicToken()
	if err != nil {
		t.Fatal(err)
	}
	second, err := randomPublicToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 64 || len(second) != 64 || first == second {
		t.Fatalf("unexpected public tokens %q %q", first, second)
	}
}
