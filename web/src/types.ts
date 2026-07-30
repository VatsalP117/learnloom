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
  concepts?: string[];
  sourceTitles?: string[];
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
}

export interface Profile {
  csrfToken?: string;
  capabilities?: { sourceDiscovery?: boolean; [key: string]: unknown };
  site?: Site | null;
  notifications?: NotificationPreferences;
}

export interface NotificationPreferences {
  configured?: boolean;
  weeklyRecap: boolean;
  reentryReminder: boolean;
  timeZone: string;
  updatedAt?: string;
}

export interface SourceValidationResult {
  status: "ready" | "unavailable";
  itemCount: number;
  message?: string;
  canonicalUrl?: string;
}

export interface SourceValidationResponse {
  sources: SourceValidationResult[];
}

export interface NewsletterCreateResponse {
  newsletter: Pick<Newsletter, "id">;
}

export interface NewsletterDetailResponse {
  newsletter: Newsletter;
  issues?: Issue[];
  lessonProgress?: WorkspaceSnapshot["lessonProgress"];
}

export interface UsernameAvailabilityResponse {
  username: string;
  available: boolean;
}

export interface SiteMutationResponse {
  site: Site;
}

export interface PublicCorrection {
  id: string;
  body: string;
  createdAt: string;
}

export interface ContentReport {
  id: string;
  category: string;
  details: string;
  status: "open" | "resolved" | "dismissed";
  resolutionReason: string;
  createdAt: string;
  resolvedAt?: string;
}

export interface ModerationAction {
  id: string;
  action: string;
  reason: string;
  createdAt: string;
}

export interface IssueModerationResponse {
  state: "clear" | "held";
  reason: string;
  corrections: PublicCorrection[];
  reports: ContentReport[];
  actions: ModerationAction[];
}

export function errorMessage(error: unknown): string {
  return error instanceof Error
    ? error.message
    : "The request could not be completed.";
}
