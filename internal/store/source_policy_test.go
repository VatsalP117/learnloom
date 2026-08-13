package store

import "testing"

func TestSourcePolicyKeysCanonicalizeExactURLAndDomain(t *testing.T) {
	t.Parallel()
	exact, domain, err := sourcePolicyKeys("HTTPS://Docs.Example.COM:443/path?q=1#section")
	if err != nil {
		t.Fatal(err)
	}
	if exact != "https://docs.example.com/path?q=1" {
		t.Fatalf("exact URL = %q", exact)
	}
	if domain != "example.com" {
		t.Fatalf("registrable domain = %q", domain)
	}
}

func TestSourcePolicyKeysPermitExactPublicIPTargets(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		raw  string
		want string
	}{
		{raw: "https://1.1.1.1:443/feed#latest", want: "https://1.1.1.1/feed"},
		{raw: "https://[2606:4700:4700::1111]/feed", want: "https://[2606:4700:4700::1111]/feed"},
	} {
		exact, domain, err := sourcePolicyKeys(test.raw)
		if err != nil {
			t.Fatalf("%s: %v", test.raw, err)
		}
		if exact != test.want || domain != "" {
			t.Fatalf("%s normalized to exact=%q domain=%q", test.raw, exact, domain)
		}
	}
}

func TestNormalizeSourcePolicyDomainRequiresRegistrableDomain(t *testing.T) {
	t.Parallel()
	if got, err := normalizeSourcePolicyValue(SourcePolicyRegistrableDomain, "Example.COM."); err != nil || got != "example.com" {
		t.Fatalf("normalized domain=%q err=%v", got, err)
	}
	for _, value := range []string{"www.example.com", "com", "1.1.1.1"} {
		if _, err := normalizeSourcePolicyValue(SourcePolicyRegistrableDomain, value); err == nil {
			t.Fatalf("non-registrable policy domain %q accepted", value)
		}
	}
}
