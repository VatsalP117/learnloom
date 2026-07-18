import { lookup } from "node:dns/promises";
import { request as httpRequest } from "node:http";
import { request as httpsRequest } from "node:https";
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
  const lookupFn = options.lookupFn ?? lookup;
  const maximumBytes = options.maximumBytes ?? DEFAULT_MAX_BYTES;
  const maximumCharacters =
    options.maximumCharacters ?? DEFAULT_MAX_CHARACTERS;
  const maximumRedirects = options.maximumRedirects ?? 3;
  const timeoutMs = options.timeoutMs ?? 15_000;
  let currentUrl = new URL(inputUrl);

  for (let redirects = 0; redirects <= maximumRedirects; redirects += 1) {
    const resolved = await resolvePublicWebUrl(currentUrl, lookupFn);
    const response = options.fetchImpl
      ? await options.fetchImpl(currentUrl, {
          method: "GET",
          redirect: "manual",
          headers: articleRequestHeaders(),
          signal: AbortSignal.timeout(timeoutMs),
        })
      : await requestPinnedArticle(resolved, {
          maximumBytes,
          timeoutMs,
          requestImpl: options.requestImpl,
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
    const body =
      response.bodyText ?? (await readBoundedText(response, maximumBytes));
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
  return (await resolvePublicWebUrl(input, lookupFn)).url;
}

async function resolvePublicWebUrl(input, lookupFn) {
  const url = input instanceof URL ? input : new URL(input);
  if (!["http:", "https:"].includes(url.protocol)) {
    throw new Error("article URL must use HTTP or HTTPS");
  }
  if (url.username || url.password) {
    throw new Error("article URL must not contain credentials");
  }
  const hostname = stripIpv6Brackets(url.hostname.toLowerCase());
  if (hostname === "localhost" || hostname.endsWith(".localhost")) {
    throw new Error("article URL resolves to a private address");
  }
  const literalVersion = isIP(hostname);
  const rawAddresses = literalVersion
    ? [{ address: hostname, family: literalVersion }]
    : await lookupFn(hostname, { all: true, verbatim: true });
  if (!Array.isArray(rawAddresses) || rawAddresses.length === 0) {
    throw new Error("article hostname did not resolve");
  }
  const addresses = rawAddresses.map(({ address, family }) => ({
    address: stripIpv6Brackets(String(address).toLowerCase()),
    family: normalizeAddressFamily(family, address),
  }));
  if (addresses.some(({ address }) => !isPublicAddress(address))) {
    throw new Error("article URL resolves to a private address");
  }
  return { url, addresses };
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
  const value = ipv6ToBigInt(address);
  if (value == null) return false;
  const mappedPrefix = 0xffffn;
  if ((value >> 32n) === mappedPrefix) {
    return isPublicIpv4(bigIntToIpv4(value & 0xffff_ffffn));
  }
  return (
    inIpv6Cidr(value, "2000::", 3) &&
    !inIpv6Cidr(value, "2001::", 23) &&
    !inIpv6Cidr(value, "2001:db8::", 32) &&
    !inIpv6Cidr(value, "2002::", 16) &&
    !inIpv6Cidr(value, "3fff::", 20)
  );
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

function requestPinnedArticle(resolved, options) {
  const { url, addresses } = resolved;
  const selected = addresses[0];
  const requestImpl =
    options.requestImpl ??
    (url.protocol === "https:" ? httpsRequest : httpRequest);
  return new Promise((resolve, reject) => {
    let settled = false;
    const fail = (error) => {
      if (settled) return;
      settled = true;
      reject(error);
    };
    const request = requestImpl(
      url,
      {
        method: "GET",
        headers: articleRequestHeaders(),
        lookup(_hostname, lookupOptions, callback) {
          if (lookupOptions?.all) {
            callback(null, [selected]);
            return;
          }
          callback(null, selected.address, selected.family);
        },
      },
      (response) => {
        const remoteAddress = stripIpv6Brackets(
          response.socket.remoteAddress?.toLowerCase() ?? "",
        );
        if (
          !isPublicAddress(remoteAddress) ||
          !addressesEquivalent(remoteAddress, selected.address)
        ) {
          response.destroy();
          fail(new Error("article connection did not use the validated address"));
          return;
        }
        const chunks = [];
        let length = 0;
        response.on("data", (chunk) => {
          length += chunk.length;
          if (length > options.maximumBytes) {
            response.destroy();
            fail(new Error("article exceeded size limit"));
            return;
          }
          chunks.push(chunk);
        });
        response.on("end", () => {
          if (settled) return;
          settled = true;
          const headers = new Map(
            Object.entries(response.headers).map(([name, value]) => [
              name.toLowerCase(),
              Array.isArray(value) ? value.join(", ") : value ?? null,
            ]),
          );
          resolve({
            status: response.statusCode ?? 0,
            ok:
              (response.statusCode ?? 0) >= 200 &&
              (response.statusCode ?? 0) < 300,
            headers: { get: (name) => headers.get(name.toLowerCase()) ?? null },
            bodyText: Buffer.concat(chunks).toString("utf8"),
          });
        });
        response.on("error", fail);
      },
    );
    request.setTimeout(options.timeoutMs, () => {
      request.destroy(new Error("article request timed out"));
    });
    request.on("error", fail);
    request.end();
  });
}

function articleRequestHeaders() {
  return {
    accept: "text/html, text/plain;q=0.9",
    "user-agent": "learnloom/0.5 (+personal learning source enrichment)",
  };
}

function normalizeAddressFamily(family, address) {
  if (family === 4 || family === "IPv4") return 4;
  if (family === 6 || family === "IPv6") return 6;
  return isIP(stripIpv6Brackets(String(address)));
}

function addressesEquivalent(left, right) {
  const leftVersion = isIP(left);
  const rightVersion = isIP(right);
  if (leftVersion !== rightVersion) return false;
  if (leftVersion === 4) return left === right;
  if (leftVersion !== 6) return false;
  return ipv6ToBigInt(left) === ipv6ToBigInt(right);
}

function stripIpv6Brackets(value) {
  return value.startsWith("[") && value.endsWith("]")
    ? value.slice(1, -1)
    : value;
}

function ipv6ToBigInt(address) {
  let normalized = stripIpv6Brackets(address.toLowerCase());
  if (normalized.includes("%")) return null;
  const ipv4Match = normalized.match(/(\d+\.\d+\.\d+\.\d+)$/);
  if (ipv4Match) {
    if (!isIP(ipv4Match[1])) return null;
    const parts = ipv4Match[1].split(".").map(Number);
    const replacement = `${((parts[0] << 8) | parts[1]).toString(16)}:${(
      (parts[2] << 8) |
      parts[3]
    ).toString(16)}`;
    normalized = `${normalized.slice(0, -ipv4Match[1].length)}${replacement}`;
  }
  const halves = normalized.split("::");
  if (halves.length > 2) return null;
  const left = halves[0] ? halves[0].split(":") : [];
  const right = halves[1] ? halves[1].split(":") : [];
  const missing = 8 - left.length - right.length;
  if (
    (halves.length === 1 && missing !== 0) ||
    (halves.length === 2 && missing < 1)
  ) {
    return null;
  }
  const groups = [...left, ...Array(Math.max(0, missing)).fill("0"), ...right];
  if (
    groups.length !== 8 ||
    groups.some((group) => !/^[\da-f]{1,4}$/.test(group))
  ) {
    return null;
  }
  return groups.reduce(
    (value, group) => (value << 16n) | BigInt(`0x${group}`),
    0n,
  );
}

function inIpv6Cidr(value, baseAddress, prefixLength) {
  const base = ipv6ToBigInt(baseAddress);
  const shift = BigInt(128 - prefixLength);
  return base != null && value >> shift === base >> shift;
}

function bigIntToIpv4(value) {
  return [24n, 16n, 8n, 0n]
    .map((shift) => Number((value >> shift) & 0xffn))
    .join(".");
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
