package source

import (
	"net/netip"
	"strings"
	"testing"
)

func TestPublicAddressPolicy(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{
		"127.0.0.1", "10.0.0.1", "169.254.169.254", "100.64.0.1",
		"192.0.2.1", "198.51.100.1", "203.0.113.1", "::1", "fc00::1",
		"fe80::1", "2001:db8::1", "::ffff:127.0.0.1",
	} {
		if isPublicAddress(netip.MustParseAddr(raw)) {
			t.Errorf("%s should be rejected", raw)
		}
	}
	for _, raw := range []string{"1.1.1.1", "8.8.8.8", "2606:4700:4700::1111"} {
		if !isPublicAddress(netip.MustParseAddr(raw)) {
			t.Errorf("%s should be public", raw)
		}
	}
}

func TestParseRSSAndAtom(t *testing.T) {
	t.Parallel()
	rss := `<rss><channel><item><title>One &amp; Two</title><link>https://example.com/a</link><description><![CDATA[<p>Hello</p>]]></description><pubDate>Sat, 18 Jul 2026 10:00:00 GMT</pubDate></item></channel></rss>`
	items, err := ParseFeed([]byte(rss), "Example")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Title != "One & Two" || items[0].Summary != "Hello" {
		t.Fatalf("unexpected RSS result: %#v", items)
	}

	atom := `<feed xmlns="http://www.w3.org/2005/Atom"><entry><title>Atom</title><link rel="alternate" href="https://example.com/b"/><summary>Body</summary><updated>2026-07-18T10:00:00Z</updated></entry></feed>`
	items, err = ParseFeed([]byte(atom), "Atom source")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].URL != "https://example.com/b" {
		t.Fatalf("unexpected Atom result: %#v", items)
	}
}

func TestValidateWebURL(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{
		"file:///etc/passwd",
		"http://user:pass@example.com",
		"http://localhost/feed",
		"http://127.0.0.1/feed",
		"http://[::1]/feed",
	} {
		if _, err := validateWebURL(raw); err == nil {
			t.Errorf("%s should be rejected", raw)
		}
	}
	if _, err := validateWebURL("https://example.com/feed.xml"); err != nil {
		t.Fatal(err)
	}
}

func TestBoundedAndCleanText(t *testing.T) {
	t.Parallel()
	if _, err := readBounded(strings.NewReader("12345"), 4); err == nil {
		t.Fatal("expected size error")
	}
	value := cleanText(`<script>bad()</script><p>Hello &amp; <b>world</b></p>`)
	if value != "bad() Hello & world" {
		t.Fatalf("unexpected clean text %q", value)
	}
}
