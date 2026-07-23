import { useCallback, useEffect, useMemo, useState } from "react";
import { apiJSON } from "./api.js";

let cachedSnapshot = null;
let snapshotRequest = null;
let issuePageRequest = null;

export function invalidateWorkspaceCache() {
  cachedSnapshot = null;
}

async function loadSnapshot(force) {
  if (cachedSnapshot && !force) return cachedSnapshot;
  if (!snapshotRequest || force) {
    snapshotRequest = apiJSON("/api/workspace")
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

export function hydrateWorkspace(workspace) {
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

export function mergeIssuePage(snapshot, page) {
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
    issuePageRequest = apiJSON(
      `/api/issues?limit=40&cursor=${encodeURIComponent(cursor)}`,
    )
      .then((page) => {
        cachedSnapshot = mergeIssuePage(cachedSnapshot, page);
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
      setError(requestError.message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const lessons = useMemo(
    () => [...(snapshot?.issues ?? [])]
      .sort((left, right) => new Date(right.createdAt) - new Date(left.createdAt)),
    [snapshot?.issues],
  );
  const reload = useCallback(() => load(true), [load]);
  const loadMore = useCallback(async () => {
    setLoadingMore(true);
    setError("");
    try {
      setSnapshot(await loadNextIssuePage());
    } catch (requestError) {
      setError(requestError.message);
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
