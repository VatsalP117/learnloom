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
  [key: string]: any;
}

export interface Review {
  issueId: string;
  objective?: string;
  questions?: string[];
  [key: string]: any;
}

export interface WorkspaceSnapshot {
  newsletters: Newsletter[];
  issues: Issue[];
  reviews?: Review[];
  nextIssueCursor?: string;
  [key: string]: any;
}

export interface Site {
  username: string;
  displayName: string;
  description?: string;
  visibility: string;
  url?: string;
  [key: string]: any;
}

export interface Profile {
  csrfToken?: string;
  capabilities?: { sourceDiscovery?: boolean; [key: string]: unknown };
  site?: Site | null;
  [key: string]: any;
}

export function errorMessage(error: unknown): string {
  return error instanceof Error
    ? error.message
    : "The request could not be completed.";
}
