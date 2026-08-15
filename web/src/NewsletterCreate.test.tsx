import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import NewsletterCreate from "./NewsletterCreate";

describe("redesigned NewsletterCreate render", () => {
  it("renders the redesigned shell with step-one validation and no effects or network", () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch");

    try {
      const markup = renderToStaticMarkup(<NewsletterCreate />);

      // The creation flow opts into the warm-neutral redesigned shell.
      expect(markup).toContain('class="atelier-app atelier-today"');

      // Step one: guided heading, progress rail, and step counter.
      expect(markup).toContain("What should become clearer?");
      expect(markup).toContain("Learning stream setup progress");
      expect(markup).toContain("Step 1 of 3");

      // Step-one gating: topic is required and Continue starts disabled.
      expect(markup).toContain('name="topic"');
      expect(markup).toContain("required");
      expect(markup).toContain("Continue");
      expect(markup).toContain('disabled=""');

      // Later steps stay hidden until gated through.
      expect(markup).not.toContain("Source policy");
      expect(markup).not.toContain("Build my learning path");

      // SSR runs no effects, so the draft restore/autosave never fetch.
      expect(fetchSpy).not.toHaveBeenCalled();
    } finally {
      fetchSpy.mockRestore();
    }
  });
});
