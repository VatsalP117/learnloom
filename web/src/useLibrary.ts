import { useCallback, useEffect, useRef, useState } from "react";
import { apiJSON } from "./api";
import { syncLessonProgress } from "./learningState";
import { errorMessage, type LibraryLesson, type LibrarySnapshot } from "./types";

export type LibraryFilter = "all" | "unread" | "in-progress" | "completed";

export function libraryQueryPath(
  query: string,
  filter: LibraryFilter,
  cursor = "",
) {
  const parameters = new URLSearchParams({
    limit: "24",
    filter,
  });
  const normalizedQuery = query.trim();
  if (normalizedQuery) parameters.set("q", normalizedQuery);
  if (cursor) parameters.set("cursor", cursor);
  return `/api/library?${parameters.toString()}`;
}

function syncLibraryProgress(lessons: LibraryLesson[]) {
  syncLessonProgress(
    lessons.flatMap((lesson) => lesson.progress ? [lesson.progress] : []),
  );
}

export function useLibrary(query: string, filter: LibraryFilter) {
  const [deferredQuery, setDeferredQuery] = useState(query);
  const [lessons, setLessons] = useState<LibraryLesson[]>([]);
  const [nextCursor, setNextCursor] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [reloadKey, setReloadKey] = useState(0);
  const requestVersion = useRef(0);

  useEffect(() => {
    const timeout = window.setTimeout(() => setDeferredQuery(query), 250);
    return () => window.clearTimeout(timeout);
  }, [query]);

  useEffect(() => {
    const controller = new AbortController();
    const version = ++requestVersion.current;
    setLoading(true);
    setLoadingMore(false);
    setError("");
    apiJSON<LibrarySnapshot>(libraryQueryPath(deferredQuery, filter), {
      signal: controller.signal,
    })
      .then((snapshot) => {
        if (version !== requestVersion.current) return;
        syncLibraryProgress(snapshot.lessons);
        setLessons(snapshot.lessons);
        setNextCursor(snapshot.nextCursor ?? "");
      })
      .catch((requestError) => {
        if (requestError instanceof Error && requestError.name === "AbortError") return;
        if (version === requestVersion.current) setError(errorMessage(requestError));
      })
      .finally(() => {
        if (version === requestVersion.current) setLoading(false);
      });
    return () => controller.abort();
  }, [deferredQuery, filter, reloadKey]);

  const loadMore = useCallback(async () => {
    if (!nextCursor || loadingMore) return;
    const version = requestVersion.current;
    setLoadingMore(true);
    setError("");
    try {
      const snapshot = await apiJSON<LibrarySnapshot>(
        libraryQueryPath(deferredQuery, filter, nextCursor),
      );
      if (version !== requestVersion.current) return;
      syncLibraryProgress(snapshot.lessons);
      setLessons((current) => {
        const known = new Set(current.map((lesson) => lesson.id));
        return [
          ...current,
          ...snapshot.lessons.filter((lesson) => !known.has(lesson.id)),
        ];
      });
      setNextCursor(snapshot.nextCursor ?? "");
    } catch (requestError) {
      if (version === requestVersion.current) setError(errorMessage(requestError));
    } finally {
      if (version === requestVersion.current) setLoadingMore(false);
    }
  }, [deferredQuery, filter, loadingMore, nextCursor]);

  const reload = useCallback(() => setReloadKey((current) => current + 1), []);

  return {
    lessons,
    error,
    loading,
    loadingMore,
    hasMore: Boolean(nextCursor),
    loadMore,
    reload,
  };
}
