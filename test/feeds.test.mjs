import assert from "node:assert/strict";
import test from "node:test";
import { fetchSources, parseFeed } from "../src/feeds.mjs";

test("parseFeed parses RSS and removes markup", () => {
  const items = parseFeed(
    `<?xml version="1.0"?><rss><channel><item>
      <title>Memory &amp; Learning</title>
      <link>https://example.com/a</link>
      <description><![CDATA[<p>Useful <strong>summary</strong>.</p>]]></description>
      <pubDate>Fri, 17 Jul 2026 08:00:00 GMT</pubDate>
    </item></channel></rss>`,
    "Test RSS",
  );
  assert.deepEqual(items, [
    {
      source: "Test RSS",
      title: "Memory & Learning",
      url: "https://example.com/a",
      summary: "Useful summary.",
      publishedAt: "2026-07-17T08:00:00.000Z",
    },
  ]);
});

test("parseFeed parses Atom href links", () => {
  const items = parseFeed(
    `<feed><entry><title>Atom item</title>
      <link rel="alternate" href="https://example.com/atom" />
      <summary>Short note</summary><updated>2026-07-18T09:00:00Z</updated>
    </entry></feed>`,
    "Test Atom",
  );
  assert.equal(items[0].url, "https://example.com/atom");
  assert.equal(items[0].summary, "Short note");
});

test("fetchSources tolerates one failed source and deduplicates items", async () => {
  const rss = `<rss><channel><item><title>A</title><link>https://example.com/a?ref=x</link>
    <description>One</description></item></channel></rss>`;
  const fetchImpl = async (url) => {
    if (url.includes("bad")) throw new Error("offline");
    return { ok: true, text: async () => rss };
  };
  const config = {
    sources: [
      { name: "Good", url: "https://good.example/feed", limit: 5 },
      { name: "Bad", url: "https://bad.example/feed", limit: 5 },
    ],
    limits: { maxItems: 10 },
  };
  const result = await fetchSources(config, fetchImpl);
  assert.equal(result.items.length, 1);
  assert.match(result.warnings[0], /offline/);
});

