import { useCallback, useEffect, useMemo, useState } from "react";
import { apiJSON } from "./api.js";

let cachedSnapshot = null;
let snapshotRequest = null;

export function invalidateWorkspaceCache() {
  cachedSnapshot = null;
}

async function loadSnapshot(force) {
  if (cachedSnapshot && !force) return cachedSnapshot;
  if (!snapshotRequest || force) {
    snapshotRequest = apiJSON("/api/workspace")
      .then((workspace) => {
        cachedSnapshot = workspace;
        return workspace;
      })
      .finally(() => {
        snapshotRequest = null;
      });
  }
  return snapshotRequest;
}

export function useWorkspace() {
  const [snapshot, setSnapshot] = useState(cachedSnapshot);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(!cachedSnapshot);

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

  return {
    snapshot,
    newsletters: snapshot?.newsletters ?? [],
    lessons,
    reviews: snapshot?.reviews ?? [],
    error,
    loading,
    reload,
  };
}
