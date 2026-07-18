import assert from "node:assert/strict";
import { EventEmitter } from "node:events";
import { PassThrough } from "node:stream";
import test from "node:test";
import {
  assertPublicWebUrl,
  enrichSourceItems,
  extractArticleMetadata,
  fetchArticleText,
} from "../src/source-enrichment.mjs";

const publicLookup = async () => [{ address: "93.184.216.34", family: 4 }];

test("extractArticleMetadata selects article text and removes active chrome", () => {
  const result = extractArticleMetadata(
    `<!doctype html><html><head>
      <link rel="canonical" href="/canonical">
      <meta name="author" content="Ada Lovelace">
      <style>.hidden{display:none}</style>
    </head><body><nav>Menu</nav><article>
      <h1>Flow control</h1>
      <p>Queues apply <strong>backpressure</strong> when consumers slow down.</p>
      <script>alert("no")</script>
    </article><footer>Footer</footer></body></html>`,
    new URL("https://example.com/original"),
  );
  assert.equal(result.canonicalUrl, "https://example.com/canonical");
  assert.equal(result.author, "Ada Lovelace");
  assert.match(result.text, /Flow control/);
  assert.match(result.text, /Queues apply backpressure/);
  assert.doesNotMatch(result.text, /Menu|alert|Footer/);
});

test("assertPublicWebUrl rejects private, credentialed, and non-web targets", async () => {
  await assert.rejects(
    assertPublicWebUrl("http://127.0.0.1/private"),
    /private address/,
  );
  await assert.rejects(
    assertPublicWebUrl("https://example.com", async () => [
      { address: "10.0.0.5", family: 4 },
    ]),
    /private address/,
  );
  await assert.rejects(
    assertPublicWebUrl("https://user:pass@example.com", publicLookup),
    /credentials/,
  );
  await assert.rejects(
    assertPublicWebUrl("file:///etc/passwd", publicLookup),
    /HTTP or HTTPS/,
  );
  for (const address of [
    "fc00::1",
    "fe80::1",
    "2001:db8::1",
    "2001::1",
    "2002:7f00:1::",
    "3fff::1",
    "::ffff:127.0.0.1",
  ]) {
    await assert.rejects(
      assertPublicWebUrl(`https://[${address}]/private`),
      /private address/,
    );
  }
});

test("default article transport pins the validated address to the socket", async () => {
  let pinnedAddress = null;
  const result = await fetchArticleText("https://rebind.example/article", {
    lookupFn: async () => [{ address: "93.184.216.34", family: 4 }],
    requestImpl(_url, options, onResponse) {
      options.lookup("rebind.example", { all: true }, (_error, addresses) => {
        const address = addresses[0].address;
        pinnedAddress = address;
      });
      const request = new EventEmitter();
      request.setTimeout = () => request;
      request.end = () => {
        const response = new PassThrough();
        response.statusCode = 200;
        response.headers = { "content-type": "text/plain" };
        response.socket = { remoteAddress: "93.184.216.34" };
        onResponse(response);
        response.end("Pinned public article content.");
      };
      request.destroy = (error) => request.emit("error", error);
      return request;
    },
  });
  assert.equal(pinnedAddress, "93.184.216.34");
  assert.equal(result.text, "Pinned public article content.");

  await assert.rejects(
    fetchArticleText("https://rebind.example/private", {
      lookupFn: publicLookup,
      requestImpl(_url, _options, onResponse) {
        const request = new EventEmitter();
        request.setTimeout = () => request;
        request.end = () => {
          const response = new PassThrough();
          response.statusCode = 200;
          response.headers = { "content-type": "text/plain" };
          response.socket = { remoteAddress: "127.0.0.1" };
          onResponse(response);
        };
        request.destroy = (error) => request.emit("error", error);
        return request;
      },
    }),
    /validated address/,
  );
});

test("fetchArticleText validates redirects and extracts bounded HTML", async () => {
  const requests = [];
  const result = await fetchArticleText("https://example.com/start", {
    lookupFn: publicLookup,
    async fetchImpl(url, options) {
      requests.push({ url: String(url), redirect: options.redirect });
      if (String(url).endsWith("/start")) {
        return new Response(null, {
          status: 302,
          headers: { location: "/article" },
        });
      }
      return new Response(
        `<html><body><main><h1>Mechanism</h1><p>${"Useful detail. ".repeat(
          80,
        )}</p></main></body></html>`,
        { headers: { "content-type": "text/html; charset=utf-8" } },
      );
    },
    maximumCharacters: 600,
  });
  assert.equal(requests.length, 2);
  assert.ok(requests.every((request) => request.redirect === "manual"));
  assert.match(result.text, /^Mechanism/);
  assert.ok(result.text.length <= 600);
});

test("fetchArticleText stops oversized and unsupported responses", async () => {
  await assert.rejects(
    fetchArticleText("https://example.com/large", {
      lookupFn: publicLookup,
      maximumBytes: 20,
      fetchImpl: async () =>
        new Response("x".repeat(100), {
          headers: { "content-type": "text/plain" },
        }),
    }),
    /size limit/,
  );
  await assert.rejects(
    fetchArticleText("https://example.com/image", {
      lookupFn: publicLookup,
      fetchImpl: async () =>
        new Response("image", {
          headers: { "content-type": "image/png" },
        }),
    }),
    /unsupported article content type/,
  );
});

test("enrichSourceItems preserves source identity and falls back independently", async () => {
  const items = [
    {
      sourceId: "S1",
      source: "Example",
      title: "Good",
      url: "https://example.com/good",
      summary: "Feed fallback one",
      publishedAt: null,
    },
    {
      sourceId: "S2",
      source: "Example",
      title: "Blocked",
      url: "https://example.com/blocked",
      summary: "Feed fallback two",
      publishedAt: null,
    },
  ];
  const enriched = await enrichSourceItems(items, {
    lookupFn: publicLookup,
    minimumCharacters: 30,
    async fetchImpl(url) {
      if (String(url).endsWith("/blocked")) {
        return new Response("no", { status: 500 });
      }
      return new Response(
        `<article><p>${"Grounded article content. ".repeat(20)}</p></article>`,
        { headers: { "content-type": "text/html" } },
      );
    },
  });
  assert.equal(enriched[0].sourceId, "S1");
  assert.equal(enriched[0].contentSource, "article");
  assert.match(enriched[0].summary, /Grounded article content/);
  assert.equal(enriched[1].contentSource, "feed-summary");
  assert.equal(enriched[1].summary, "Feed fallback two");
  assert.match(enriched[1].enrichmentError, /HTTP 500/);
});
