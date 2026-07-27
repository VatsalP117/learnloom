import { describe, expect, it } from "vitest";
import {
  hydrateWorkspace,
  mergeIssuePage,
  workspaceSnapshotIsFresh,
} from "./useWorkspace";

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

describe("workspace freshness", () => {
  it("keeps recent snapshots instant and expires old snapshots", () => {
    const now = 1_000_000;
    expect(workspaceSnapshotIsFresh(now - 60_000, now)).toBe(true);
    expect(workspaceSnapshotIsFresh(now - 5 * 60_000, now)).toBe(false);
    expect(workspaceSnapshotIsFresh(0, now)).toBe(false);
  });
});
