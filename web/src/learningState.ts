const stateKey = "learnloom.learning-state.v1";

interface LessonProgress {
  progress: number;
  completed: boolean;
  lastOpenedAt?: string;
  completedAt?: string;
}

interface ReviewProgress {
  status: string;
  reviewedAt?: string;
}

interface LearningState {
  lessons?: Record<string, Partial<LessonProgress>>;
  reviews?: Record<string, Partial<ReviewProgress>>;
}

function readState(): LearningState {
  try {
    return JSON.parse(window.localStorage.getItem(stateKey) ?? "{}");
  } catch {
    return {};
  }
}

function writeState(state: LearningState) {
  window.localStorage.setItem(stateKey, JSON.stringify(state));
  window.dispatchEvent(new CustomEvent("learnloom:state"));
}

export function lessonState(issueId: string): LessonProgress {
  return {
    progress: 0,
    completed: false,
    ...readState().lessons?.[issueId],
  };
}

export function updateLessonState(issueId: string, patch: Partial<LessonProgress>) {
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
export function reviewState(reviewId: string): ReviewProgress {
  return { status: "due", ...readState().reviews?.[reviewId] };
}

export function updateReviewState(reviewId: string, patch: Partial<ReviewProgress>) {
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
