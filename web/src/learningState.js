const stateKey = "learnloom.learning-state.v1";

function readState() {
  try {
    return JSON.parse(window.localStorage.getItem(stateKey) ?? "{}");
  } catch {
    return {};
  }
}

function writeState(state) {
  window.localStorage.setItem(stateKey, JSON.stringify(state));
  window.dispatchEvent(new CustomEvent("learnloom:state"));
}

export function lessonState(issueId) {
  return readState().lessons?.[issueId] ?? { progress: 0, completed: false };
}

export function updateLessonState(issueId, patch) {
  const state = readState();
  const current = state.lessons?.[issueId] ?? {};
  writeState({
    ...state,
    lessons: {
      ...state.lessons,
      [issueId]: { ...current, ...patch },
    },
  });
}
export function reviewState(reviewId) {
  return readState().reviews?.[reviewId] ?? { status: "due" };
}

export function updateReviewState(reviewId, patch) {
  const state = readState();
  const current = state.reviews?.[reviewId] ?? {};
  writeState({
    ...state,
    reviews: {
      ...state.reviews,
      [reviewId]: { ...current, ...patch },
    },
  });
}
