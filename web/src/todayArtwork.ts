// Deterministic artwork assignment for the redesigned Today page.
//
// Each learning stream maps to one artwork through a stable hash of its
// stream ID, so a stream keeps the same artwork across reloads without
// any schema or API change. Cards use small card-sized assets; the hero
// uses larger hero-sized assets, so below-the-fold art never fetches
// hero-sized files.
//
// Artwork is decorative: images ship with empty alt text and never carry
// information.

import art01Card from "./assets/today/art-01-card.webp";
import art01Hero from "./assets/today/art-01-hero.webp";
import art02Card from "./assets/today/art-02-card.webp";
import art02Hero from "./assets/today/art-02-hero.webp";
import art03Card from "./assets/today/art-03-card.webp";
import art03Hero from "./assets/today/art-03-hero.webp";
import art04Card from "./assets/today/art-04-card.webp";
import art04Hero from "./assets/today/art-04-hero.webp";
import art05Card from "./assets/today/art-05-card.webp";
import art05Hero from "./assets/today/art-05-hero.webp";
import art06Card from "./assets/today/art-06-card.webp";
import art06Hero from "./assets/today/art-06-hero.webp";
import art07Card from "./assets/today/art-07-card.webp";
import art07Hero from "./assets/today/art-07-hero.webp";
import art08Card from "./assets/today/art-08-card.webp";
import art08Hero from "./assets/today/art-08-hero.webp";

export interface TodayArtwork {
  /** Stable key of the artwork (also the derived asset file stem). */
  id: string;
  /** Compact card-sized asset (~680px wide). */
  card: string;
  /** Responsive candidates for the card asset. */
  cardSrcSet: string;
  /** Large hero-sized asset (~1440px wide). */
  hero: string;
  /** Responsive candidates for the hero asset. */
  heroSrcSet: string;
}

function asset(id: string, card: string, hero: string): TodayArtwork {
  return {
    id,
    card,
    cardSrcSet: `${card} 680w`,
    hero,
    heroSrcSet: `${hero} 1440w`,
  };
}

export const todayArtworks: TodayArtwork[] = [
  asset("art-01", art01Card, art01Hero),
  asset("art-02", art02Card, art02Hero),
  asset("art-03", art03Card, art03Hero),
  asset("art-04", art04Card, art04Hero),
  asset("art-05", art05Card, art05Hero),
  asset("art-06", art06Card, art06Hero),
  asset("art-07", art07Card, art07Hero),
  asset("art-08", art08Card, art08Hero),
];

/**
 * Neutral fallback used when no stream ID exists (for example a review
 * hero with no owning lesson). Chosen as the quietest, lowest-saturation
 * artwork so it stays calm inside the white shell.
 */
export const todayFallbackArtwork = todayArtworks[5];

/** FNV-1a 32-bit hash kept stable for a given stream ID. */
export function artworkIndexForStream(streamId: string): number {
  let hash = 0x811c9dc5;
  for (let index = 0; index < streamId.length; index += 1) {
    hash ^= streamId.charCodeAt(index);
    hash = Math.imul(hash, 0x01000193) >>> 0;
  }
  return hash % todayArtworks.length;
}

/**
 * Deterministic artwork for a stream. A missing or empty stream ID maps
 * to the neutral fallback instead of failing or picking at random.
 */
export function artworkForStream(streamId?: string | null): TodayArtwork {
  if (!streamId) return todayFallbackArtwork;
  return todayArtworks[artworkIndexForStream(streamId)];
}
