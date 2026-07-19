package httpapp

import "testing"

func TestClassifyHost(t *testing.T) {
	t.Parallel()
	cases := map[string]HostKind{
		"learnloom.blog":         HostApex,
		"www.learnloom.blog":     HostWWW,
		"app.learnloom.blog:443": HostApp,
		"vatsal.learnloom.blog":  HostSite,
	}
	for value, wanted := range cases {
		got, err := ClassifyHost(value, "learnloom.blog")
		if err != nil || got.Kind != wanted {
			t.Errorf("%q: got=%#v err=%v", value, got, err)
		}
	}
	for _, value := range []string{
		"evil.example", "a.b.learnloom.blog", "api.learnloom.blog",
		"vatsal.learnloom.blog.evil.example", "learnloom.blog.",
	} {
		if _, err := ClassifyHost(value, "learnloom.blog"); err == nil {
			t.Errorf("%q should be rejected", value)
		}
	}
}
