import { useCallback, useEffect, useMemo, useState } from "react";
import { apiJSON } from "./api.js";

export function useWorkspace({ includeDossiers = false, includeDetails = true } = {}) {
  const [snapshot, setSnapshot] = useState(null);
  const [details, setDetails] = useState([]);
  const [dossiers, setDossiers] = useState({});
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const workspace = await apiJSON("/api/newsletters");
      const streamDetails = includeDetails
        ? await Promise.all(
          (workspace.newsletters ?? []).map((newsletter) =>
            apiJSON(`/api/newsletters/${encodeURIComponent(newsletter.id)}`),
          ),
        )
        : [];
      setSnapshot(workspace);
      setDetails(streamDetails);

      if (includeDossiers) {
        const generated = streamDetails
          .flatMap((detail) => detail.issues ?? [])
          .filter((issue) => issue.status === "generated")
          .slice(0, 8);
        const issueDetails = await Promise.all(
          generated.map((issue) =>
            apiJSON(`/api/issues/${encodeURIComponent(issue.id)}`)
              .then((value) => [issue.id, value])
              .catch(() => [issue.id, null]),
          ),
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
    reload: load,
  };
}
