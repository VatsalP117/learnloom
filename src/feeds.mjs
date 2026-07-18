const XML_ENTITIES = new Map([
  ["amp", "&"],
  ["lt", "<"],
  ["gt", ">"],
  ["quot", '"'],
  ["apos", "'"],
]);

export function parseFeed(xml, sourceName = "Unknown source") {
  if (typeof xml !== "string" || xml.trim() === "") {
    return [];
  }

  const rssBlocks = blocks(xml, "item");
  if (rssBlocks.length > 0) {
    return rssBlocks.map((block) => parseRssItem(block, sourceName)).filter(hasTitleAndLink);
  }

  return blocks(xml, "entry")
    .map((block) => parseAtomEntry(block, sourceName))
    .filter(hasTitleAndLink);
}

export async function fetchSources(config, fetchImpl = globalThis.fetch) {
  const outcomes = await Promise.allSettled(
    config.sources.map(async (source) => {
      const response = await fetchImpl(source.url, {
        headers: {
          accept: "application/atom+xml, application/rss+xml, application/xml, text/xml",
          "user-agent": "learnloom/0.1 (+local personal reader)",
        },
        signal: AbortSignal.timeout(20_000),
      });
      if (!response.ok) {
        throw new Error(`${source.name} returned HTTP ${response.status}`);
      }
      const xml = await response.text();
      return parseFeed(xml, source.name).slice(0, source.limit);
    }),
  );

  const errors = [];
  const items = [];
  outcomes.forEach((outcome, index) => {
    if (outcome.status === "fulfilled") {
      items.push(...outcome.value);
    } else {
      errors.push(`${config.sources[index].name}: ${outcome.reason.message}`);
    }
  });

  if (items.length === 0) {
    throw new Error(`No feed items could be loaded. ${errors.join("; ")}`);
  }

  const unique = deduplicate(items)
    .sort((left, right) => dateValue(right.publishedAt) - dateValue(left.publishedAt))
    .slice(0, config.limits.maxItems);

  return { items: unique, warnings: errors };
}

function parseRssItem(block, sourceName) {
  return {
    source: sourceName,
    title: textOf(block, ["title"]),
    url: textOf(block, ["link", "guid"]),
    summary: cleanText(textOf(block, ["content:encoded", "description", "content"])),
    publishedAt: normalizeDate(textOf(block, ["pubDate", "dc:date", "date"])),
  };
}

function parseAtomEntry(block, sourceName) {
  const linkTag = firstTag(block, "link");
  const href = linkTag?.match(/\bhref\s*=\s*["']([^"']+)["']/i)?.[1] ?? "";
  return {
    source: sourceName,
    title: textOf(block, ["title"]),
    url: decodeXml(href || textOf(block, ["link", "id"])),
    summary: cleanText(textOf(block, ["summary", "content"])),
    publishedAt: normalizeDate(textOf(block, ["published", "updated"])),
  };
}

function blocks(xml, tag) {
  const expression = new RegExp(`<${tag}\\b[^>]*>([\\s\\S]*?)<\\/${tag}>`, "gi");
  return [...xml.matchAll(expression)].map((match) => match[1]);
}

function textOf(xml, names) {
  for (const name of names) {
    const escaped = name.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    const expression = new RegExp(
      `<${escaped}\\b[^>]*>([\\s\\S]*?)<\\/${escaped}>`,
      "i",
    );
    const match = xml.match(expression);
    if (match) {
      return decodeXml(match[1].replace(/^<!\[CDATA\[([\s\S]*)\]\]>$/i, "$1")).trim();
    }
  }
  return "";
}

function firstTag(xml, name) {
  const expression = new RegExp(`<${name}\\b[^>]*\\/?>`, "i");
  return xml.match(expression)?.[0] ?? null;
}

function decodeXml(value) {
  return value.replace(
    /&(#x[\da-f]+|#\d+|amp|lt|gt|quot|apos);/gi,
    (_, entity) => {
      if (entity[0] === "#") {
        const hexadecimal = entity[1].toLowerCase() === "x";
        const number = Number.parseInt(entity.slice(hexadecimal ? 2 : 1), hexadecimal ? 16 : 10);
        return Number.isFinite(number) ? String.fromCodePoint(number) : _;
      }
      return XML_ENTITIES.get(entity.toLowerCase()) ?? _;
    },
  );
}

function cleanText(value) {
  return decodeXml(
    value
      .replace(/<script\b[^>]*>[\s\S]*?<\/script>/gi, " ")
      .replace(/<style\b[^>]*>[\s\S]*?<\/style>/gi, " ")
      .replace(/<[^>]+>/g, " "),
  )
    .replace(/\s+/g, " ")
    .replace(/\s+([,.;:!?])/g, "$1")
    .trim();
}

function normalizeDate(value) {
  if (!value) return null;
  const date = new Date(value);
  return Number.isNaN(date.valueOf()) ? null : date.toISOString();
}

function hasTitleAndLink(item) {
  return Boolean(item.title && item.url);
}

function deduplicate(items) {
  const seen = new Set();
  return items.filter((item) => {
    const key = `${item.url.replace(/[#?].*$/, "")}\n${item.title.toLowerCase()}`;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}

function dateValue(value) {
  return value ? new Date(value).valueOf() || 0 : 0;
}
