export interface Source {
  name: string;
  url: string;
  limit?: number;
}

export interface Newsletter {
  id: string;
  name: string;
  topic: string;
  learnerLevel?: string;
  learnerGoal?: string;
  lessonMinutes?: number;
  scheduleTime?: string;
  timeZone?: string;
  active?: boolean;
  emailEnabled?: boolean;
  aiExplorationEnabled?: boolean;
  siteVisible?: boolean;
  sources?: Source[];
  issueCount?: number;
  generatedCount?: number;
  sentCount?: number;
  [key: string]: any;
}

export interface Issue {
  id: string;
  newsletterId?: string;
  newsletter?: Newsletter;
  title?: string;
  status?: string;
  publicationState?: string;
  createdAt?: string;
  error?: string;
  failureCode?: string;
  failureCategory?: string;
  failureStage?: string;
  failureRetryable?: boolean;
  incidentId?: string;
  [key: string]: any;
}

export interface Review {
  id: string;
  issueId: string;
  objective: string;
  prompt: string;
  answerRubric: string;
  correctiveExplanation: string;
  stage: number;
  dueAt: string;
  lastReviewedAt?: string;
  [key: string]: any;
}

export interface WorkspaceSnapshot {
  newsletters: Newsletter[];
  issues: Issue[];
  reviews?: Review[];
  lessonProgress?: Array<{
    issueId: string;
    progress: number;
    completedAt?: string;
    updatedAt?: string;
  }>;
  nextIssueCursor?: string;
  retention?: RetentionState;
  [key: string]: any;
}

export interface RetentionState {
  activatedAt?: string;
  returnedAfterSevenDays: boolean;
  lastActivityAt?: string;
  inactive: boolean;
  daysAway: number;
  actionLabel?: string;
  actionUrl?: string;
}

export interface LibraryLesson extends Issue {
  newsletter: Newsletter;
  progress?: {
    issueId: string;
    progress: number;
    completedAt?: string;
    updatedAt?: string;
  };
}

export interface LibrarySnapshot {
  lessons: LibraryLesson[];
  nextCursor?: string;
}

export interface Site {
  username: string;
  displayName: string;
  description?: string;
  visibility: string;
  searchIndexing: boolean;
  url?: string;
  [key: string]: any;
}

export interface Profile {
  csrfToken?: string;
  capabilities?: { sourceDiscovery?: boolean; [key: string]: unknown };
  site?: Site | null;
  notifications?: NotificationPreferences;
  [key: string]: any;
}

export interface NotificationPreferences {
  configured?: boolean;
  weeklyRecap: boolean;
  reentryReminder: boolean;
  timeZone: string;
  updatedAt?: string;
}

export function errorMessage(error: unknown): string {
  return error instanceof Error
    ? error.message
    : "The request could not be completed.";
}
