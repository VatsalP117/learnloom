import { lookup } from "node:dns/promises";
import { isIP } from "node:net";

const DEFAULT_MAX_BYTES = 512 * 1024;
const DEFAULT_MAX_CHARACTERS = 16_000;
const REDIRECT_STATUSES = new Set([301, 302, 303, 307, 308]);

export async function enrichSourceItems(items, options = {}) {
  if (!Array.isArray(items) || items.length === 0) {
    throw new Error("Source enrichment requires at least one Source Item.");
  }
  return Promise.all(
    items.map(async (item) => {
      try {
        const enriched = await fetchArticleText(item.url, options);
        if (enriched.text.length < (options.minimumCharacters ?? 400)) {
          throw new Error("article text was too short");
        }
        return {
          ...item,
          summary: enriched.text,
          contentSource: "article",
          canonicalUrl: enriched.canonicalUrl ?? item.url,
          author: enriched.author,
          enrichmentError: null,
        };
      } catch (error) {
        return {
          ...item,
          contentSource: "feed-summary",
          canonicalUrl: item.url,
          author: null,
          enrichmentError: safeEnrichmentError(error),
        };
      }
    }),
  );
}

export async function fetchArticleText(inputUrl, options = {}) {
  const fetchImpl = options.fetchImpl ?? globalThis.fetch;
  const lookupFn = options.lookupFn ?? lookup;
  const maximumBytes = options.maximumBytes ?? DEFAULT_MAX_BYTES;
  const maximumCharacters =
    options.maximumCharacters ?? DEFAULT_MAX_CHARACTERS;
  const maximumRedirects = options.maximumRedirects ?? 3;
  const timeoutMs = options.timeoutMs ?? 15_000;
  let currentUrl = new URL(inputUrl);

  for (let redirects = 0; redirects <= maximumRedirects; redirects += 1) {
    await assertPublicWebUrl(currentUrl, lookupFn);
    const response = await fetchImpl(currentUrl, {
      method: "GET",
      redirect: "manual",
      headers: {
        accept: "text/html, text/plain;q=0.9",
        "user-agent": "learnloom/0.5 (+personal learning source enrichment)",
      },
      signal: AbortSignal.timeout(timeoutMs),
    });

    if (REDIRECT_STATUSES.has(response.status)) {
      if (redirects === maximumRedirects) {
        throw new Error("article redirected too many times");
      }
      const location = response.headers.get("location");
      if (!location) throw new Error("article redirect had no location");
      currentUrl = new URL(location, currentUrl);
      continue;
    }
    if (!response.ok) {
      throw new Error(`article returned HTTP ${response.status}`);
    }
    const contentType = response.headers
      .get("content-type")
      ?.toLowerCase() ?? "";
    if (
      !contentType.startsWith("text/html") &&
      !contentType.startsWith("text/plain")
    ) {
      throw new Error(`unsupported article content type ${contentType || "unknown"}`);
    }
    const body = await readBoundedText(response, maximumBytes);
    const metadata =
      contentType.startsWith("text/html")
        ? extractArticleMetadata(body, currentUrl)
        : { text: body, canonicalUrl: currentUrl.toString(), author: null };
    return {
      ...metadata,
      text: truncate(metadata.text, maximumCharacters),
    };
  }
  throw new Error("article enrichment could not complete");
}

export async function assertPublicWebUrl(input, lookupFn = lookup) {
  const url = input instanceof URL ? input : new URL(input);
  if (!["http:", "https:"].includes(url.protocol)) {
    throw new Error("article URL must use HTTP or HTTPS");
  }
  if (url.username || url.password) {
    throw new Error("article URL must not contain credentials");
  }
  const hostname = url.hostname.toLowerCase();
  if (hostname === "localhost" || hostname.endsWith(".localhost")) {
    throw new Error("article URL resolves to a private address");
  }
  const literalVersion = isIP(hostname);
  const addresses = literalVersion
    ? [{ address: hostname, family: literalVersion }]
    : await lookupFn(hostname, { all: true, verbatim: true });
  if (!Array.isArray(addresses) || addresses.length === 0) {
    throw new Error("article hostname did not resolve");
  }
  if (addresses.some(({ address }) => !isPublicAddress(address))) {
    throw new Error("article URL resolves to a private address");
  }
  return url;
}

export function extractArticleMetadata(html, pageUrl) {
  const canonical = firstMatch(
    html,
    /<link\b[^>]*\brel\s*=\s*["'][^"']*\bcanonical\b[^"']*["'][^>]*\bhref\s*=\s*["']([^"']+)["'][^>]*>/i,
  ) ?? firstMatch(
    html,
    /<link\b[^>]*\bhref\s*=\s*["']([^"']+)["'][^>]*\brel\s*=\s*["'][^"']*\bcanonical\b[^"']*["'][^>]*>/i,
  );
  const author =
    firstMatch(
      html,
      /<meta\b[^>]*(?:name|property)\s*=\s*["'](?:author|article:author)["'][^>]*content\s*=\s*["']([^"']+)["'][^>]*>/i,
    ) ??
    firstMatch(
      html,
      /<meta\b[^>]*content\s*=\s*["']([^"']+)["'][^>]*(?:name|property)\s*=\s*["'](?:author|article:author)["'][^>]*>/i,
    );
  const primary =
    firstMatch(html, /<article\b[^>]*>([\s\S]*?)<\/article>/i) ??
    firstMatch(html, /<main\b[^>]*>([\s\S]*?)<\/main>/i) ??
    firstMatch(html, /<body\b[^>]*>([\s\S]*?)<\/body>/i) ??
    html;
  const text = htmlToText(primary);
  let canonicalUrl = pageUrl.toString();
  if (canonical) {
    try {
      const parsed = new URL(decodeHtml(canonical), pageUrl);
      if (["http:", "https:"].includes(parsed.protocol)) {
        canonicalUrl = parsed.toString();
      }
    } catch {
      // Keep the fetched URL when canonical metadata is malformed.
    }
  }
  return {
    text,
    canonicalUrl,
    author: author ? decodeHtml(author).trim().slice(0, 300) : null,
  };
}

function htmlToText(value) {
  return decodeHtml(
    value
      .replace(/<!--[\s\S]*?-->/g, " ")
      .replace(
        /<(script|style|noscript|svg|nav|header|footer|form)\b[^>]*>[\s\S]*?<\/\1>/gi,
        " ",
      )
      .replace(/<(br|p|div|section|article|main|h[1-6]|li|blockquote)\b[^>]*>/gi, "\n")
      .replace(/<[^>]+>/g, " "),
  )
    .replace(/\r/g, "")
    .replace(/[ \t]+/g, " ")
    .replace(/ *\n */g, "\n")
    .replace(/\n{3,}/g, "\n\n")
    .trim();
}

async function readBoundedText(response, maximumBytes) {
  if (!response.body?.getReader) {
    const buffer = Buffer.from(await response.arrayBuffer());
    if (buffer.length > maximumBytes) throw new Error("article exceeded size limit");
    return buffer.toString("utf8");
  }
  const reader = response.body.getReader();
  const chunks = [];
  let length = 0;
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      length += value.byteLength;
      if (length > maximumBytes) {
        await reader.cancel();
        throw new Error("article exceeded size limit");
      }
      chunks.push(value);
    }
  } finally {
    reader.releaseLock();
  }
  return Buffer.concat(chunks.map((chunk) => Buffer.from(chunk))).toString(
    "utf8",
  );
}

function isPublicAddress(address) {
  const version = isIP(address);
  if (version === 4) return isPublicIpv4(address);
  if (version !== 6) return false;
  const normalized = address.toLowerCase();
  if (normalized.startsWith("::ffff:")) {
    return isPublicIpv4(normalized.slice(7));
  }
  if (
    normalized === "::" ||
    normalized === "::1" ||
    normalized.startsWith("fc") ||
    normalized.startsWith("fd") ||
    /^fe[89ab]/.test(normalized) ||
    normalized.startsWith("ff") ||
    normalized.startsWith("2001:db8:")
  ) {
    return false;
  }
  return true;
}

function isPublicIpv4(address) {
  const parts = address.split(".").map(Number);
  if (
    parts.length !== 4 ||
    parts.some((part) => !Number.isInteger(part) || part < 0 || part > 255)
  ) {
    return false;
  }
  const [first, second, third] = parts;
  if (
    first === 0 ||
    first === 10 ||
    first === 127 ||
    first >= 224 ||
    (first === 100 && second >= 64 && second <= 127) ||
    (first === 169 && second === 254) ||
    (first === 172 && second >= 16 && second <= 31) ||
    (first === 192 && second === 168) ||
    (first === 198 && (second === 18 || second === 19)) ||
    (first === 192 && second === 0 && (third === 0 || third === 2)) ||
    (first === 198 && second === 51 && third === 100) ||
    (first === 203 && second === 0 && third === 113)
  ) {
    return false;
  }
  return true;
}

function decodeHtml(value) {
  const named = new Map([
    ["amp", "&"],
    ["lt", "<"],
    ["gt", ">"],
    ["quot", '"'],
    ["apos", "'"],
    ["nbsp", " "],
  ]);
  return value.replace(
    /&(#x[\da-f]+|#\d+|amp|lt|gt|quot|apos|nbsp);/gi,
    (match, entity) => {
      if (entity.startsWith("#")) {
        const hexadecimal = entity[1]?.toLowerCase() === "x";
        const code = Number.parseInt(
          entity.slice(hexadecimal ? 2 : 1),
          hexadecimal ? 16 : 10,
        );
        return Number.isFinite(code) ? String.fromCodePoint(code) : match;
      }
      return named.get(entity.toLowerCase()) ?? match;
    },
  );
}

function firstMatch(value, expression) {
  return value.match(expression)?.[1] ?? null;
}

function truncate(value, maximum) {
  if (value.length <= maximum) return value;
  return `${value.slice(0, maximum - 16).trimEnd()}\n[truncated]`;
}

function safeEnrichmentError(error) {
  return String(error?.message ?? "article enrichment failed")
    .replace(/\s+/g, " ")
    .slice(0, 240);
}

