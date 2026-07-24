import { useCallback, useEffect, useMemo, useState } from "react";
import { apiJSON } from "./api";
import { errorMessage, type WorkspaceSnapshot } from "./types";

let cachedSnapshot: WorkspaceSnapshot | null = null;
let snapshotRequest: Promise<WorkspaceSnapshot> | null = null;
let issuePageRequest: Promise<WorkspaceSnapshot | null> | null = null;

export function invalidateWorkspaceCache(_newsletterId?: string) {
  cachedSnapshot = null;
}

async function loadSnapshot(force: boolean) {
  if (cachedSnapshot && !force) return cachedSnapshot;
  if (!snapshotRequest || force) {
    snapshotRequest = apiJSON<WorkspaceSnapshot>("/api/workspace")
      .then((workspace) => {
        cachedSnapshot = hydrateWorkspace(workspace);
        return cachedSnapshot;
      })
      .finally(() => {
        snapshotRequest = null;
      });
  }
  return snapshotRequest;
}

export function hydrateWorkspace(workspace: WorkspaceSnapshot): WorkspaceSnapshot {
  const newslettersByID = new Map(
    (workspace.newsletters ?? []).map((newsletter) => [newsletter.id, newsletter]),
  );
  return {
    ...workspace,
    issues: (workspace.issues ?? []).map((issue) => ({
      ...issue,
      newsletter: newslettersByID.get(issue.newsletterId),
    })),
  };
}

export function mergeIssuePage(
  snapshot: WorkspaceSnapshot,
  page: Pick<WorkspaceSnapshot, "issues" | "nextIssueCursor">,
): WorkspaceSnapshot {
  const hydrated = hydrateWorkspace({
    newsletters: snapshot.newsletters,
    issues: page.issues,
  });
  const known = new Set(snapshot.issues.map((issue) => issue.id));
  return {
    ...snapshot,
    issues: [
      ...snapshot.issues,
      ...hydrated.issues.filter((issue) => !known.has(issue.id)),
    ],
    nextIssueCursor: page.nextIssueCursor,
  };
}

async function loadNextIssuePage() {
  const cursor = cachedSnapshot?.nextIssueCursor;
  if (!cursor) return cachedSnapshot;
  if (!issuePageRequest) {
    issuePageRequest = apiJSON<Pick<WorkspaceSnapshot, "issues" | "nextIssueCursor">>(
      `/api/issues?limit=40&cursor=${encodeURIComponent(cursor)}`,
    )
      .then((page) => {
          if (cachedSnapshot) cachedSnapshot = mergeIssuePage(cachedSnapshot, page);
        return cachedSnapshot;
      })
      .finally(() => {
        issuePageRequest = null;
      });
  }
  return issuePageRequest;
}

export function useWorkspace() {
  const [snapshot, setSnapshot] = useState(cachedSnapshot);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(!cachedSnapshot);
  const [loadingMore, setLoadingMore] = useState(false);

  const load = useCallback(async (force = false) => {
    setLoading(force || !cachedSnapshot);
    setError("");
    try {
      setSnapshot(await loadSnapshot(force));
    } catch (requestError) {
      setError(errorMessage(requestError));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const lessons = useMemo(
    () => [...(snapshot?.issues ?? [])]
      .sort((left, right) =>
        new Date(right.createdAt).getTime() - new Date(left.createdAt).getTime()
      ),
    [snapshot?.issues],
  );
  const reload = useCallback(() => load(true), [load]);
  const loadMore = useCallback(async () => {
    setLoadingMore(true);
    setError("");
    try {
      setSnapshot(await loadNextIssuePage());
    } catch (requestError) {
      setError(errorMessage(requestError));
    } finally {
      setLoadingMore(false);
    }
  }, []);

  return {
    snapshot,
    newsletters: snapshot?.newsletters ?? [],
    lessons,
    reviews: snapshot?.reviews ?? [],
    error,
    loading,
    loadingMore,
    hasMore: Boolean(snapshot?.nextIssueCursor),
    loadMore,
    reload,
  };
}
