export interface LessonRetrievalResponse {
  issueId: string;
  promptKey: string;
  response?: string;
  skipped: boolean;
  revealedAt?: string;
  updatedAt: string;
}

export interface RetrievalResponseState {
  response: string;
  skipped: boolean;
  revealed: boolean;
  busy: boolean;
  saving: boolean;
  error: string;
}

export function resumeLessonProgress(localProgress = 0, serverProgress = 0) {
  return Math.max(0, Math.min(100, Math.max(localProgress, serverProgress)));
}

export function canRevealRetrieval(response = "") {
  return response.trim().length >= 3;
}

export function initialRetrievalState(responses: LessonRetrievalResponse[] = []) {
  return Object.fromEntries(responses.map((response) => [response.promptKey, {
    response: response.response ?? "",
    skipped: Boolean(response.skipped),
    revealed: Boolean(response.revealedAt),
    busy: false,
    saving: false,
    error: "",
  }])) as Record<string, RetrievalResponseState>;
}
