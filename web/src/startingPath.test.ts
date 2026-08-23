import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  clearSourceDossierIntent,
  parseSourceDossierHint,
  pendingSourceDossierHint,
  pendingSourceDossierReturnURL,
  rememberSourceDossierIntent,
  resolveInitialOnboardingDraft,
} from "./startingPath";

const VALID = "dossier-30000000-0000-0000-0000-000000000000";
const ENCODED = "%64%6f%73%73%69%65%72%2d" + "30000000-0000-0000-0000-000000000000";

function storage(): Storage {
  const values = new Map<string, string>();
  return {
    getItem: (key) => values.get(key) ?? null,
    setItem: (key, value) => void values.set(key, value),
    removeItem: (key) => void values.delete(key),
    clear: () => values.clear(),
    key: (index) => [...values.keys()][index] ?? null,
    get length() {
      return values.size;
    },
  };
}

describe("parseSourceDossierHint", () => {
  it("accepts only the strict dossier-<uuid> shape", () => {
    expect(parseSourceDossierHint(`?source_dossier=${VALID}`)).toEqual({
      dossierId: VALID,
      returnUrl: `/newsletters/new?source_dossier=${encodeURIComponent(VALID)}`,
    });
    expect(parseSourceDossierHint(`?utm_source=x&source_dossier=${VALID}`)?.dossierId)
      .toBe(VALID);
    expect(parseSourceDossierHint(`?source_dossier=${VALID}&utm_source=x`)?.dossierId)
      .toBe(VALID);
  });

  it("rejects arbitrary URLs and malformed shapes", () => {
    for (const search of [
      "",
      "?",
      "?source_dossier=",
      "?source_dossier=30000000-0000-0000-0000-000000000000",
      "?source_dossier=other-30000000-0000-0000-0000-000000000000",
      "?source_dossier=dossier-nope",
      "?source_dossier=dossier-",
      "?source_dossier=https://evil.example.com/phish",
      "?source_dossier=//evil.example.com/phish",
      "?source_dossier=javascript:alert(1)",
    ]) {
      expect(parseSourceDossierHint(search), search).toBeNull();
    }
  });
});

describe("pending source dossier intent", () => {
  let original: Storage | undefined;

  beforeEach(() => {
    original = globalThis.sessionStorage;
    Object.defineProperty(globalThis, "sessionStorage", {
      configurable: true,
      value: storage(),
    });
  });

  afterEach(() => {
    Object.defineProperty(globalThis, "sessionStorage", {
      configurable: true,
      value: original,
    });
  });

  it("remembers a valid hint and returns the fixed return URL", () => {
    expect(pendingSourceDossierReturnURL()).toBeNull();
    rememberSourceDossierIntent(`?source_dossier=${VALID}`);
    expect(pendingSourceDossierHint()).toEqual({
      dossierId: VALID,
      returnUrl: `/newsletters/new?source_dossier=${encodeURIComponent(VALID)}`,
    });
    expect(pendingSourceDossierReturnURL()).toBe(
      `/newsletters/new?source_dossier=${encodeURIComponent(VALID)}`,
    );
  });

  it("ignores invalid hints without disturbing normal navigation", () => {
    rememberSourceDossierIntent("?next=https://evil.example.com");
    expect(pendingSourceDossierHint()).toBeNull();
    expect(pendingSourceDossierReturnURL()).toBeNull();
  });

  it("replaces an earlier intent with a newer valid hint", () => {
    rememberSourceDossierIntent(`?source_dossier=${VALID}`);
    const other = "dossier-40000000-0000-0000-0000-000000000000";
    rememberSourceDossierIntent(`?source_dossier=${other}`);
    expect(pendingSourceDossierHint()?.dossierId).toBe(other);
  });

  it("clears the intent only on explicit clear", () => {
    rememberSourceDossierIntent(`?source_dossier=${VALID}`);
    clearSourceDossierIntent();
    expect(pendingSourceDossierHint()).toBeNull();
    expect(pendingSourceDossierReturnURL()).toBeNull();
  });

  it("ignores malformed stored values", () => {
    sessionStorage.setItem("learnloom.pending.sourceDossier", "https://evil.example.com");
    expect(pendingSourceDossierHint()).toBeNull();
    sessionStorage.setItem("learnloom.pending.sourceDossier", ENCODED);
    expect(pendingSourceDossierHint()).toBeNull();
  });
});

describe("resolveInitialOnboardingDraft", () => {
  type Draft = { id: string };
  const existing: Draft = { id: "existing-draft" };
  const seeded: Draft = { id: "seeded-draft" };
  let original: Storage | undefined;

  beforeEach(() => {
    original = globalThis.sessionStorage;
    Object.defineProperty(globalThis, "sessionStorage", {
      configurable: true,
      value: storage(),
    });
  });

  afterEach(() => {
    Object.defineProperty(globalThis, "sessionStorage", {
      configurable: true,
      value: original,
    });
  });

  it("restores the existing draft unchanged and never calls the seed endpoint", async () => {
    const startSeeded = vi.fn().mockResolvedValue({ draft: seeded });
    rememberSourceDossierIntent(`?source_dossier=${VALID}`);
    const resolution = await resolveInitialOnboardingDraft({
      getDraft: vi.fn().mockResolvedValue({ draft: existing }),
      startSeeded,
      search: "",
    });
    expect(resolution).toEqual({ draft: existing, seeded: false, error: null });
    expect(startSeeded).not.toHaveBeenCalled();
    expect(pendingSourceDossierHint()).toBeNull();
  });

  it("seeds a fresh draft from a pending start intent and consumes it", async () => {
    rememberSourceDossierIntent(`?source_dossier=${VALID}`);
    const resolution = await resolveInitialOnboardingDraft({
      getDraft: vi.fn().mockResolvedValue({ draft: null }),
      startSeeded: vi.fn().mockResolvedValue({ draft: seeded }),
      search: "",
    });
    expect(resolution).toEqual({ draft: seeded, seeded: true, error: null });
    expect(pendingSourceDossierHint()).toBeNull();
  });

  it("remembers a hint from the current URL before seeding", async () => {
    const resolution = await resolveInitialOnboardingDraft({
      getDraft: vi.fn().mockResolvedValue({ draft: null }),
      startSeeded: vi.fn().mockResolvedValue({ draft: seeded }),
      search: `?source_dossier=${VALID}`,
    });
    expect(resolution.draft).toEqual(seeded);
    expect(resolution.seeded).toBe(true);
    expect(pendingSourceDossierHint()).toBeNull();
  });

  it("falls back to blank onboarding and consumes the intent on seed failure", async () => {
    rememberSourceDossierIntent(`?source_dossier=${VALID}`);
    const failure = Object.assign(new Error("not found"), { code: "not_found", status: 404 });
    const resolution = await resolveInitialOnboardingDraft({
      getDraft: vi.fn().mockResolvedValue({ draft: null }),
      startSeeded: vi.fn().mockRejectedValue(failure),
      search: "",
    });
    expect(resolution).toEqual({ draft: null, seeded: true, error: failure });
    expect(pendingSourceDossierHint()).toBeNull();
  });

  it("falls back to blank onboarding without attempting a seed when no hint exists", async () => {
    const startSeeded = vi.fn();
    const resolution = await resolveInitialOnboardingDraft({
      getDraft: vi.fn().mockResolvedValue({ draft: null }),
      startSeeded,
      search: "",
    });
    expect(resolution).toEqual({ draft: null, seeded: false, error: null });
    expect(startSeeded).not.toHaveBeenCalled();
  });

  it("reports a failed initial draft read without attempting a seed", async () => {
    const failure = new Error("network");
    const resolution = await resolveInitialOnboardingDraft({
      getDraft: vi.fn().mockRejectedValue(failure),
      startSeeded: vi.fn(),
      search: "",
    });
    expect(resolution).toEqual({ draft: null, seeded: false, error: failure });
  });
});
