// Navigation intent for "Start a path like this".
//
// The only trusted hint shape is a `source_dossier=dossier-<uuid>` query
// parameter: it is remembered on the app origin (sessionStorage) and turned
// back into the fixed `/newsletters/new?source_dossier=...` return URL after
// authentication. Arbitrary URLs or malformed IDs are never accepted, and the
// server derives the actual attribution from the referral cookie, so this
// hint is purely a navigation helper.

const DOSSIER_ID_PATTERN =
  /^dossier-([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$/i;

const STORAGE_KEY = "learnloom.pending.sourceDossier";

export interface SourceDossierHint {
  /** Normalized `dossier-<uuid>` public Dossier ID. */
  dossierId: string;
  /** Fixed app-origin return URL carrying the encoded hint. */
  returnUrl: string;
}

/** Parses a strict `source_dossier=dossier-<uuid>` hint from a search string. */
export function parseSourceDossierHint(search: string): SourceDossierHint | null {
  const params = new URLSearchParams(search);
  const match = DOSSIER_ID_PATTERN.exec(params.get("source_dossier")?.trim() ?? "");
  if (!match) return null;
  const dossierId = `dossier-${match[1].toLowerCase()}`;
  return {
    dossierId,
    returnUrl: `/newsletters/new?source_dossier=${encodeURIComponent(dossierId)}`,
  };
}

/**
 * Remembers a valid navigation intent on the app origin. Invalid or missing
 * hints are ignored and never disturb existing intents.
 */
export function rememberSourceDossierIntent(search: string): void {
  const hint = parseSourceDossierHint(search);
  if (!hint) return;
  try {
    sessionStorage.setItem(STORAGE_KEY, hint.dossierId);
  } catch {
    // Storage can be unavailable (private mode); navigation simply continues
    // to the default destination.
  }
}

/** Reads back the pending hint, validating its stored shape. */
export function pendingSourceDossierHint(): SourceDossierHint | null {
  let stored = "";
  try {
    stored = sessionStorage.getItem(STORAGE_KEY) ?? "";
  } catch {
    return null;
  }
  if (stored === "") return null;
  const hint = parseSourceDossierHint(`?source_dossier=${encodeURIComponent(stored)}`);
  return hint?.dossierId === stored ? hint : null;
}

/** Returns the fixed return URL when a valid pending intent exists. */
export function pendingSourceDossierReturnURL(): string | null {
  return pendingSourceDossierHint()?.returnUrl ?? null;
}

/** Clears the pending intent (called only after the seed attempt). */
export function clearSourceDossierIntent(): void {
  try {
    sessionStorage.removeItem(STORAGE_KEY);
  } catch {
    // Nothing to clear.
  }
}

export interface InitialDraftResolution<Draft> {
  /** Restored draft, either the existing one or the freshly seeded one. */
  draft: Draft | null;
  /** True when the seed endpoint was attempted (intent consumed). */
  seeded: boolean;
  /** Last failure, when the draft could not be resolved at all. */
  error: unknown;
}

/**
 * Resolves the initial onboarding draft control flow:
 * - an existing draft wins cleanly and the seed endpoint is never called;
 * - with no draft and a valid pending start intent, the authenticated seed
 *   endpoint is attempted exactly once and the intent is then consumed
 *   (success, 404/409, or network failure all clear it);
 * - anything else falls back to a blank onboarding draft.
 */
export async function resolveInitialOnboardingDraft<Draft extends { id: string }>({
  getDraft,
  startSeeded,
  search,
}: {
  getDraft: () => Promise<{ draft: Draft | null }>;
  startSeeded: () => Promise<{ draft: Draft | null }>;
  search: string;
}): Promise<InitialDraftResolution<Draft>> {
  let draftResponse;
  try {
    draftResponse = await getDraft();
  } catch (error) {
    return { draft: null, seeded: false, error };
  }
  if (draftResponse.draft) {
    // The existing draft wins without being replaced. Consume any pending
    // public-path intent so future authentication does not keep redirecting
    // the learner back to creation indefinitely.
    rememberSourceDossierIntent(search);
    clearSourceDossierIntent();
    return { draft: draftResponse.draft, seeded: false, error: null };
  }
  rememberSourceDossierIntent(search);
  if (!pendingSourceDossierHint()) {
    return { draft: null, seeded: false, error: null };
  }
  try {
    const response = await startSeeded();
    return { draft: response.draft, seeded: true, error: null };
  } catch (error) {
    return { draft: null, seeded: true, error };
  } finally {
    clearSourceDossierIntent();
  }
}
