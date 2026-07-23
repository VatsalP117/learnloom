import { useCallback, useEffect, useMemo, useState } from "react";
import { apiJSON } from "./api.js";

let cachedSnapshot = null;
let snapshotRequest = null;
const cachedDetails = new Map();
const detailRequests = new Map();
const cachedDossiers = new Map();
const dossierRequests = new Map();

export function invalidateWorkspaceCache(newsletterId) {
  cachedSnapshot = null;
  if (newsletterId) {
    cachedDetails.delete(newsletterId);
  } else {
    cachedDetails.clear();
    cachedDossiers.clear();
  }
}

function cachedWorkspace(includeDetails, includeDossiers) {
  const newsletters = cachedSnapshot?.newsletters ?? [];
  const details = includeDetails
    ? newsletters.map((newsletter) => cachedDetails.get(newsletter.id)).filter(Boolean)
    : [];
  const generated = includeDossiers
    ? details
      .flatMap((detail) => detail.issues ?? [])
      .filter((issue) => issue.status === "generated")
      .slice(0, 8)
    : [];
  const dossiers = Object.fromEntries(
    generated
      .map((issue) => [issue.id, cachedDossiers.get(issue.id)])
      .filter(([, value]) => value),
  );
  const complete = Boolean(cachedSnapshot) &&
    (!includeDetails || details.length === newsletters.length) &&
    (!includeDossiers || Object.keys(dossiers).length === generated.length);

  return { snapshot: cachedSnapshot, details, dossiers, complete };
}

async function loadSnapshot(force) {
  if (cachedSnapshot && !force) return cachedSnapshot;
  if (!snapshotRequest || force) {
    snapshotRequest = apiJSON("/api/newsletters")
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

async function loadDetail(newsletterId, force) {
  if (cachedDetails.has(newsletterId) && !force) return cachedDetails.get(newsletterId);
  if (!detailRequests.has(newsletterId) || force) {
    const request = apiJSON(`/api/newsletters/${encodeURIComponent(newsletterId)}`)
      .then((detail) => {
        cachedDetails.set(newsletterId, detail);
        return detail;
      })
      .finally(() => detailRequests.delete(newsletterId));
    detailRequests.set(newsletterId, request);
  }
  return detailRequests.get(newsletterId);
}

async function loadDossier(issueId, force) {
  if (cachedDossiers.has(issueId) && !force) return cachedDossiers.get(issueId);
  if (!dossierRequests.has(issueId) || force) {
    const request = apiJSON(`/api/issues/${encodeURIComponent(issueId)}`)
      .then((dossier) => {
        cachedDossiers.set(issueId, dossier);
        return dossier;
      })
      .catch(() => null)
      .finally(() => dossierRequests.delete(issueId));
    dossierRequests.set(issueId, request);
  }
  return dossierRequests.get(issueId);
}

export function useWorkspace({ includeDossiers = false, includeDetails = true } = {}) {
  const initial = cachedWorkspace(includeDetails, includeDossiers);
  const [snapshot, setSnapshot] = useState(initial.snapshot);
  const [details, setDetails] = useState(initial.details);
  const [dossiers, setDossiers] = useState(initial.dossiers);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(!initial.complete);

  const load = useCallback(async (force = false) => {
    const existing = cachedWorkspace(includeDetails, includeDossiers);
    setLoading(force || !existing.complete);
    setError("");
    try {
      const workspace = await loadSnapshot(force);
      setSnapshot(workspace);

      const streamDetails = includeDetails
        ? await Promise.all(
          (workspace.newsletters ?? []).map((newsletter) =>
            loadDetail(newsletter.id, force),
          ),
        )
        : [];
      setDetails(streamDetails);

      if (includeDossiers) {
        const generated = streamDetails
          .flatMap((detail) => detail.issues ?? [])
          .filter((issue) => issue.status === "generated")
          .slice(0, 8);
        const issueDetails = await Promise.all(
          generated.map(async (issue) => [issue.id, await loadDossier(issue.id, force)]),
        );
        setDossiers(Object.fromEntries(issueDetails.filter(([, value]) => value)));
      }
    } catch (requestError) {
      setError(requestError.message);
    } finally {
      setLoading(false);
    }
  }, [includeDetails, includeDossiers]);

  useEffect(() => {
    load();
  }, [load]);

  const reload = useCallback(() => load(true), [load]);

  const lessons = useMemo(
    () =>
      details
        .flatMap((detail) =>
          (detail.issues ?? []).map((issue) => ({
            ...issue,
            newsletter: detail.newsletter,
          })),
        )
        .sort((left, right) => new Date(right.createdAt) - new Date(left.createdAt)),
    [details],
  );

  return {
    snapshot,
    newsletters: snapshot?.newsletters ?? [],
    details,
    lessons,
    dossiers,
    error,
    loading,
    reload,
  };
}
