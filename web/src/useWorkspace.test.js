import { describe, expect, it } from "vitest";
import { hydrateWorkspace, mergeIssuePage } from "./useWorkspace.js";

const newsletter = {
  id: "stream-1",
  name: "Systems",
  topic: "How systems change",
};

describe("workspace issue pagination", () => {
  it("hydrates compact issue records with their newsletter", () => {
    const workspace = hydrateWorkspace({
      newsletters: [newsletter],
      issues: [{ id: "issue-1", newsletterId: newsletter.id }],
    });

    expect(workspace.issues[0].newsletter).toEqual(newsletter);
  });

  it("appends unique older issues and advances the cursor", () => {
    const snapshot = hydrateWorkspace({
      newsletters: [newsletter],
      issues: [{ id: "issue-1", newsletterId: newsletter.id }],
      nextIssueCursor: "page-2",
    });
    const merged = mergeIssuePage(snapshot, {
      issues: [
        { id: "issue-1", newsletterId: newsletter.id },
        { id: "issue-2", newsletterId: newsletter.id },
      ],
      nextIssueCursor: "",
    });

    expect(merged.issues.map((issue) => issue.id)).toEqual(["issue-1", "issue-2"]);
    expect(merged.issues[1].newsletter).toEqual(newsletter);
    expect(merged.nextIssueCursor).toBe("");
  });
});
