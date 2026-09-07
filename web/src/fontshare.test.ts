import { describe, expect, it } from "vitest";
import { loadProductStylesheet } from "./fontshare";

type FakeLink = {
  rel: string;
  href: string;
  attributes: Record<string, string>;
  setAttribute(name: string, value: string): void;
};

function fakeDocument() {
  const links: FakeLink[] = [];
  const doc = {
    head: {
      append(link: FakeLink) {
        links.push(link);
      },
    },
    querySelector: (selector: string) =>
      selector === "link[data-product-stylesheet]" && links.length > 0
        ? links[0]
        : null,
    createElement: (_tag: string) => ({
      rel: "",
      href: "",
      attributes: {} as Record<string, string>,
      setAttribute(name: string, value: string) {
        this.attributes[name] = value;
      },
    }),
  };
  return { doc: doc as unknown as Document, links };
}

describe("loadProductStylesheet", () => {
  it("does nothing when no document exists (node renders)", () => {
    expect(loadProductStylesheet(undefined)).toBe(false);
  });

  it("injects the Satoshi stylesheet once", () => {
    const { doc, links } = fakeDocument();
    expect(loadProductStylesheet(doc)).toBe(true);

    const [link] = links;
    expect(links).toHaveLength(1);
    expect(link.rel).toBe("stylesheet");
    expect(link.href).toContain("api.fontshare.com/v2/css");
    expect(link.href).toContain("satoshi@400,500,700");
  });

  it("dedupes repeated injections", () => {
    const { doc, links } = fakeDocument();
    expect(loadProductStylesheet(doc)).toBe(true);
    expect(loadProductStylesheet(doc)).toBe(false);
    expect(links).toHaveLength(1);
  });
});