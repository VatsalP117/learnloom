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
  sourceMode?: "discovered" | "provided" | "hybrid";
  sourceReviewMode?: "auto" | "review";
  sourceApprovedAt?: string;
  scheduleTime?: string;
  timeZone?: string;
  rhythmMode?: "evidence_led" | "daily" | "selected_weekdays" | "weekly_synthesis";
  selectedWeekdays?: number[];
  effectiveRhythmMode?: "evidence_led" | "daily" | "selected_weekdays" | "weekly_synthesis";
  autoThrottleEnabled?: boolean;
  unopenedLessonLimit?: number;
  rhythmReason?: string;
  rhythmThrottledAt?: string;
  active?: boolean;
  emailEnabled?: boolean;
  aiExplorationEnabled?: boolean;
  siteVisible?: boolean;
  sources?: Source[];
  issueCount?: number;
  generatedCount?: number;
  sentCount?: number;
  capabilityCount?: number;
  recalledCapabilityCount?: number;
  currentGapCount?: number;
  lessonPublicationDefault?: "draft" | "published";
  lessonPublicationDefaultReviewedAt?: string;
}

export interface Issue {
  id: string;
  newsletterId?: string;
  newsletter?: Newsletter;
  title?: string;
  status?: string;
  publicationState?: string;
  publicationUpdatedAt?: string;
  firstPublishReviewedAt?: string;
  publishedAt?: string;
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
  todayFocus?: TodayFocus;
}

export interface TodayFocus {
  kind: "lesson" | "review" | "reentry" | "clear";
  subjectId: string;
  newsletterId?: string;
  title?: string;
  newsletterName?: string;
  lessonMinutes?: number;
  progress?: number;
  dueCount?: number;
  reasonCode: string;
  reason: string;
  actionLabel: string;
  actionUrl: string;
  score: number;
  components: Record<string, number>;
  selectedAt: string;
}

export interface RetentionState {
  activatedAt?: string;
  returnedAfterSevenDays: boolean;
  lastActivityAt?: string;
  inactive: boolean;
  daysAway: number;
  actionLabel?: string;
  actionUrl?: string;
  reentryNewsletterId?: string;
  reentryNewsletterName?: string;
}

export interface LibraryLesson {
  id: string;
  title: string;
  createdAt: string;
  newsletter: {
    name: string;
    lessonMinutes: number;
  };
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
}

export interface PublicGrowthAnalytics {
  periodDays: number;
  views: number;
  uniqueViewers: number;
  shares: number;
	follows: number;
  ctaClicks: number;
	attributedSignups: number;
  attributedActivations: number;
  publishedDossiers: number;
}

export interface PublicGrowthAnalyticsResponse {
  analytics: PublicGrowthAnalytics;
}

export interface BillingEntitlement {
  planId: "free" | "pro";
  planName: string;
  subscriptionStatus: "free" | "trialing" | "active" | "past_due" | "paused" | "canceled" | "refunded";
  entitlementStatus: "active" | "grace" | "generation_paused";
  generationAllowance: number;
  generationUsed: number;
  generationRemaining: number;
  periodStart: string;
  periodEnd: string;
  trialEndsAt?: string;
  graceEndsAt?: string;
  cancelAtPeriodEnd: boolean;
  canGenerate: boolean;
}

export interface BillingEntitlementResponse {
  billing: BillingEntitlement;
  commerceAvailable: boolean;
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

export interface SourcePortfolioPreviewItem {
  title: string;
  url: string;
  registrableDomain: string;
  role: "official_primary" | "research" | "practitioner_explainer" | "reporting_context" | "counterweight";
  selectionReason: string;
}

export interface SourcePortfolioPreviewResponse {
  rankingVersion: string;
  items: SourcePortfolioPreviewItem[];
  missingRoles: SourcePortfolioPreviewItem["role"][];
  warnings: number;
  researchPlan: {
    initialConcepts: string[];
    likelyFirstLesson: string;
    objective: string;
    minimumPreparationMinutes: number;
    maximumPreparationMinutes: number;
  };
}

export interface SourceCatalogItem {
  id: string;
  displayName: string;
  canonicalUrl: string;
  origin: "provided" | "discovered";
  scope: string;
  kind?: string;
  state: string;
  health: string;
  role?: "official_primary" | "research" | "practitioner_explainer" | "reporting_context" | "counterweight";
  rankingVersion?: string;
  preference: "neutral" | "preferred" | "blocked";
  discoveryReason?: string;
}

export interface NewsletterCreateResponse {
  newsletter: Pick<Newsletter, "id">;
}

export interface OnboardingDraftResponse {
  draft: null | {
    id: string;
    step: number;
    revision: number;
    updatedAt: string;
    payload: {
      name?: string;
      topic?: string;
      learnerLevel?: string;
      learnerGoal?: string;
      lessonMinutes?: number;
      scheduleTime?: string;
      timeZone?: string;
      active?: boolean;
      emailEnabled?: boolean;
      aiExplorationEnabled?: boolean;
      sourceMode?: "discovered" | "provided" | "hybrid";
      sourceReviewMode?: "auto" | "review";
      sources?: Source[];
      templateId?: string;
      templateVersion?: number;
    };
  };
}

export interface NewsletterDetailResponse {
  newsletter: Newsletter;
  site?: Site | null;
  resendConfigured?: boolean;
  issues?: Issue[];
  lessonProgress?: WorkspaceSnapshot["lessonProgress"];
  sourceSummary?: {
    healthy: number;
    needsAttention: number;
  };
  sourceCatalog?: SourceCatalogItem[];
  curriculum?: {
    outcome?: string;
    concepts?: Array<{
      key: string;
      label: string;
      confidenceScore: number;
      completedCount: number;
    }>;
    milestones?: Array<{
      key: string;
      label: string;
      statement: string;
      stage: "explained" | "retrieved" | "recalled_solidly";
      completedCount: number;
      reviewAttemptCount: number;
      confidenceScore: number;
    }>;
    currentGaps?: Array<{
      key: string;
      label: string;
      reason: string;
    }>;
    recall?: {
      dueCount: number;
      practicedConcepts: number;
      solidConcepts: number;
      summary: string;
    };
    suggestedNextConcepts?: string[];
    timeline?: Array<{
      issueId: string;
      title: string;
      concepts: string[];
    }>;
  };
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
