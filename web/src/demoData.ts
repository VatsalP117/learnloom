import type { APIRequestOptions } from "./api";

const createdAt = (daysAgo: number, hour = 8) => {
  const value = new Date();
  value.setDate(value.getDate() - daysAgo);
  value.setHours(hour, 12, 0, 0);
  return value.toISOString();
};

let demoOnboardingDraft = null;
let demoCreatedNewsletter: any = null;
let demoCreatedIssue: any = null;
const demoProgress: Record<string, { progress: number; completedAt?: string; updatedAt: string }> = {};
const demoRetrievals: Record<string, Array<{
  issueId: string;
  promptKey: string;
  response?: string;
  skipped: boolean;
  revealedAt?: string;
  updatedAt: string;
}>> = {};
const demoModeration: Record<string, {
  state: "clear" | "held";
  reason: string;
  corrections: Array<{ id: string; body: string; createdAt: string }>;
  reports: any[];
  actions: any[];
}> = {};

const newsletters = [
  {
    id: "ai-evaluation",
    name: "Production AI Evaluation",
    topic: "How to evaluate AI systems against real tasks, risks, and release decisions.",
    learnerLevel: "intermediate",
    learnerGoal: "Design an evaluation practice that catches costly failures before an AI feature ships.",
    lessonMinutes: 12,
    sources: [
      { name: "NIST AI RMF", url: "https://airc.nist.gov/airmf-resources/airmf/5-sec-core/", limit: 6 },
      { name: "OpenAI Evals", url: "https://platform.openai.com/docs/guides/evals", limit: 6 },
      { name: "LLM-as-a-Judge research", url: "https://arxiv.org/abs/2306.05685", limit: 5 },
    ],
    scheduleTime: "08:00",
    timeZone: "Asia/Kolkata",
    active: true,
    emailEnabled: true,
    emailRecipients: ["alex@example.com"],
    aiExplorationEnabled: true,
    publicSlug: "production-ai-evaluation",
    siteVisible: true,
    issueCount: 14,
    generatedCount: 14,
    sentCount: 14,
  },
  {
    id: "intelligence",
    name: "Reliable RAG Systems",
    topic: "Retrieval quality, grounding, evaluation, and failure diagnosis for production RAG.",
    learnerLevel: "advanced",
    learnerGoal: "Separate retrieval failures from generation failures and improve each with evidence.",
    lessonMinutes: 12,
    sources: [
      { name: "OpenAI Retrieval", url: "https://platform.openai.com/docs/guides/retrieval", limit: 5 },
      { name: "Azure AI Search", url: "https://learn.microsoft.com/en-us/azure/search/retrieval-augmented-generation-overview", limit: 5 },
      { name: "RAG paper", url: "https://arxiv.org/abs/2005.11401", limit: 5 },
    ],
    scheduleTime: "07:30",
    timeZone: "Asia/Kolkata",
    active: true,
    emailEnabled: true,
    emailRecipients: ["alex@example.com"],
    aiExplorationEnabled: false,
    publicSlug: "reliable-rag-systems",
    siteVisible: true,
    issueCount: 21,
    generatedCount: 20,
    sentCount: 20,
  },
  {
    id: "agents",
    name: "Reliable AI Agents",
    topic: "Agent architecture, tool boundaries, trajectory evaluation, and production oversight.",
    learnerLevel: "intermediate",
    learnerGoal: "Build agents that complete useful work without hiding control, security, or reliability failures.",
    lessonMinutes: 12,
    sources: [
      { name: "Anthropic: Building effective agents", url: "https://www.anthropic.com/engineering/building-effective-agents", limit: 6 },
      { name: "OpenAI Agents guide", url: "https://platform.openai.com/docs/guides/agents", limit: 5 },
      { name: "ReAct paper", url: "https://arxiv.org/abs/2210.03629", limit: 5 },
    ],
    scheduleTime: "18:00",
    timeZone: "Asia/Kolkata",
    active: false,
    emailEnabled: false,
    emailRecipients: [],
    aiExplorationEnabled: false,
    publicSlug: "reliable-ai-agents",
    siteVisible: true,
    issueCount: 12,
    generatedCount: 12,
    sentCount: 8,
  },
];

const issueTitles = [
  "Why polished output is not reliable output",
  "Build an evaluation set from real failure costs",
  "Turn a rubric into an inspectable release rule",
  "Calibrate model judges against human decisions",
  "Monitor evaluation drift after release",
];

const issuesByNewsletter = Object.fromEntries(
  newsletters.map((newsletter, newsletterIndex) => [
    newsletter.id,
    issueTitles.map((title, index) => ({
      id: `${newsletter.id}-issue-${index + 1}`,
      title:
        newsletterIndex === 0
          ? title
          : [
              "When a model learns a shortcut",
              "Why benchmarks stop measuring what matters",
              "The hidden cost of fluent answers",
              "How feedback changes a system",
              "What uncertainty should look like",
            ][index],
      trigger: index === 1 ? "manual" : "scheduled",
      scheduledLocalDate: createdAt(index).slice(0, 10),
      status: "generated",
      publicationState: index === 4 ? "private" : "published",
      createdAt: createdAt(index),
      delivery: newsletter.emailEnabled
        ? {
            status: "delivered",
            attemptCount: 1,
            createdAt: createdAt(index),
            completedAt: createdAt(index, 8),
          }
        : null,
    })),
  ]),
);

const dossierByIssue: Record<string, {
  readTime: number;
  deck: string;
  objective: string;
  sections: Array<{
    label: string;
    heading: string;
    paragraphs: string[];
    callout?: string;
  }>;
  retrieval: string[];
  application: string;
}> = {
  "ai-evaluation-issue-1": {
    readTime: 12,
    deck:
      "A fluent AI answer can still be stale, unsupported, or unsafe. Turn quality from an impression into a decision you can inspect.",
    objective:
      "Distinguish a persuasive AI output from one that has passed representative cases, explicit criteria, and a release threshold.",
    sections: [
      {
        label: "The mechanism",
        heading: "A demo proves possibility. An evaluation estimates reliability.",
        paragraphs: [
          "A product demo usually selects a cooperative prompt and rewards an impressive response. Production sends ambiguous evidence, outdated documents, adversarial instructions, and requests whose mistakes have unequal costs.",
          "An evaluation connects representative cases to atomic criteria and then to a decision: release, revise, block, or escalate. The score matters only because the decision rule gives it operational meaning.",
        ],
        callout:
          "Evaluation = cases × criteria × decision rule. A leaderboard number without a consequence is only a measurement.",
      },
      {
        label: "A worked example",
        heading: "Build twelve cases around the work and its failure costs.",
        paragraphs: [
          "For an AI research brief, combine routine, ambiguous, high-cost, and adversarial cases. Write the expected behavior and expensive failure before selecting a metric. This keeps the test anchored to actual use.",
          "Score checkable criteria independently: every material claim is grounded, the evidence version is current, uncertainty is visible, and unresolved contradictions trigger escalation. A single quality score would hide which contract failed.",
        ],
      },
      {
        label: "Skeptical review",
        heading: "The model explains a lot. It does not explain everything.",
        paragraphs: [
          "Twelve cases can reveal useful failures, but they cannot establish general safety. A model judge can add scale, yet order effects and correlated model errors mean it cannot serve as unquestioned ground truth.",
          "Blind irrelevant presentation details, human-label an audit slice, swap pairwise answer order, and route disagreement or high-cost cases to people. Expand the set with meaningful production failures.",
        ],
      },
    ],
    retrieval: [
      "Why does a benchmark score become useful only when it connects to a decision?",
      "What four kinds of cases would expose more than a polished demo?",
      "When must a model-judge verdict route to a human?",
    ],
    application:
      "Choose an AI feature you know. Write one high-cost failure case, two binary success criteria, and the exact condition that blocks release or requires human review.",
  },
};

export const demoSite = {
  username: "alex",
  displayName: "Alex’s AI Engineering Notes",
  description: "Source-grounded lessons on evaluating and operating production AI systems.",
  visibility: "public",
  searchIndexing: true,
  url: "https://alex.learnloom.blog",
};

export function demoResponse(path: string, options: APIRequestOptions = {}) {
  const method = (options.method ?? "GET").toUpperCase();
  const requestBody = options.body as Record<string, any> | undefined;
  const requestURL = new URL(path, "https://demo.learnloom.local");

  if (requestURL.pathname === "/api/library" && method === "GET") {
    const query = (requestURL.searchParams.get("q") ?? "").trim().toLowerCase();
    const filter = requestURL.searchParams.get("filter") ?? "all";
    const progressByIssue: Record<string, {
      issueId: string;
      progress: number;
      completedAt?: string;
      updatedAt: string;
    }> = {
      "ai-evaluation-issue-1": {
        issueId: "ai-evaluation-issue-1",
        progress: 42,
        updatedAt: createdAt(0),
      },
      "ai-evaluation-issue-2": {
        issueId: "ai-evaluation-issue-2",
        progress: 100,
        completedAt: createdAt(1),
        updatedAt: createdAt(1),
      },
    };
    const libraryLessons = newsletters
      .flatMap((newsletter) =>
        (issuesByNewsletter[newsletter.id] ?? []).map((issue) => ({
          lesson: {
            id: issue.id,
            title: issue.title,
            createdAt: issue.createdAt,
            newsletter: {
              name: newsletter.name,
              lessonMinutes: newsletter.lessonMinutes,
            },
            progress: progressByIssue[issue.id],
          },
          searchText: `${issue.title} ${newsletter.name} ${newsletter.topic}`.toLowerCase(),
        })),
      )
    const lessons = libraryLessons
      .filter(({ lesson, searchText }) => {
        const progress = lesson.progress;
        const matchesFilter =
          filter === "all" ||
          (filter === "completed" && Boolean(progress?.completedAt)) ||
          (filter === "in-progress" && Boolean(progress?.progress && !progress.completedAt)) ||
          (filter === "unread" && !progress?.progress);
        return matchesFilter && searchText.includes(query);
      })
      .map(({ lesson }) => lesson);
    return { lessons, nextCursor: "" };
  }

  if (path === "/api/workspace" && method === "GET") {
    const issues = newsletters.flatMap((newsletter) =>
      (issuesByNewsletter[newsletter.id] ?? []).map((issue) => ({
        ...issue,
        newsletterId: newsletter.id,
      })),
    );
    const reviews = issues
      .filter((issue) => issue.status === "generated")
      .slice(0, 8)
      .flatMap((issue) => {
        const dossier = dossierByIssue[issue.id] ?? dossierByIssue["ai-evaluation-issue-1"];
        return dossier.retrieval.slice(0, 1).map((prompt, index) => ({
          id: `${issue.id}-review-${index + 1}`,
          issueId: issue.id,
          objective: dossier.objective,
          prompt,
          answerRubric: "Name the mechanism, the evidence, and an important limit.",
          correctiveExplanation: "Reopen the lesson and trace the mechanism once more.",
          stage: 0,
          dueAt: createdAt(0),
        }));
      });
    return {
      summary: {
        newsletters: newsletters.length,
        active: newsletters.filter((item) => item.active).length,
        generated: newsletters.reduce((total, item) => total + item.generatedCount, 0),
      },
      newsletters,
      issues,
      nextIssueCursor: "",
      reviews,
    };
  }

  if (path === "/api/newsletters" && method === "GET") {
    return {
      summary: {
        newsletters: newsletters.length,
        active: newsletters.filter((item) => item.active).length,
        generated: newsletters.reduce((total, item) => total + item.generatedCount, 0),
      },
      newsletters,
    };
  }

  if (path === "/api/newsletters" && method === "POST") {
    demoCreatedNewsletter = {
      id: "new-learning-path",
      name: requestBody?.name || requestBody?.topic || "New learning path",
      topic: requestBody?.topic || "A new subject",
      learnerLevel: requestBody?.learnerLevel || "intermediate",
      learnerGoal: requestBody?.learnerGoal || "Build a practical understanding.",
      lessonMinutes: requestBody?.lessonMinutes || 12,
      sourceMode: requestBody?.sourceMode || "discovered",
      sourceReviewMode: requestBody?.sourceReviewMode || "auto",
      sources: requestBody?.sources || [],
      scheduleTime: requestBody?.scheduleTime || "08:00",
      timeZone: requestBody?.timeZone || "UTC",
      active: true,
      emailEnabled: Boolean(requestBody?.emailEnabled),
      aiExplorationEnabled: false,
      siteVisible: false,
      issueCount: 1,
      generatedCount: 0,
      sentCount: 0,
    };
    demoCreatedIssue = {
      id: "new-learning-path-issue-1",
      status: "queued",
      publicationState: "private",
      createdAt: new Date().toISOString(),
    };
    demoOnboardingDraft = null;
    return { newsletter: demoCreatedNewsletter, issue: demoCreatedIssue };
  }

  if (path === "/api/onboarding/draft" && method === "GET") {
    return { draft: demoOnboardingDraft };
  }

  if (path === "/api/onboarding/draft" && method === "PUT") {
    demoOnboardingDraft = {
        id: requestBody?.draftId ?? demoOnboardingDraft?.id ?? "demo-onboarding",
        step: requestBody?.step ?? 1,
        revision: (demoOnboardingDraft?.revision ?? 0) + 1,
      payload: requestBody?.payload ?? {},
      updatedAt: new Date().toISOString(),
    };
    return { draft: demoOnboardingDraft };
  }

  if (path === "/api/onboarding/draft" && method === "DELETE") {
    demoOnboardingDraft = null;
    return {};
  }

  if (path === "/api/sources/validate" && method === "POST") {
    return {
      sources: (requestBody?.sources ?? []).map((source) => ({
        status: "ready",
        itemCount: 8,
        canonicalUrl: source.url,
      })),
    };
  }

  if (path === "/api/source-portfolio/preview" && method === "POST") {
    const topic = requestBody?.topic ?? "your topic";
    return {
      rankingVersion: "source-rank-v2",
      warnings: 0,
      missingRoles: [],
      researchPlan: {
        initialConcepts: [
          `Foundations and boundaries of ${topic}`,
          "Core mechanisms and causal relationships",
          "Evidence quality and counterarguments",
          requestBody?.learnerGoal
            ? `Apply the model to: ${requestBody.learnerGoal}`
            : "Practical applications and failure modes",
        ],
        likelyFirstLesson: `Build a working model of ${topic}`,
        objective: "Explain the core mechanisms, test the evidence, and recognize important limitations.",
        minimumPreparationMinutes: 5,
        maximumPreparationMinutes: 15,
      },
      items: [
        {
          title: `Primary institutions working on ${topic}`,
          url: "https://www.nist.gov/",
          registrableDomain: "nist.gov",
          role: "official_primary",
          selectionReason: "Primary institutional material with stable ownership and direct subject authority.",
        },
        {
          title: `Current research evidence for ${topic}`,
          url: "https://www.nature.com/",
          registrableDomain: "nature.com",
          role: "research",
          selectionReason: "Research-led evidence that can test mechanisms and important claims.",
        },
        {
          title: `How practitioners apply ${topic}`,
          url: "https://spectrum.ieee.org/",
          registrableDomain: "ieee.org",
          role: "practitioner_explainer",
          selectionReason: "A technically grounded explanation connecting evidence to real practice.",
        },
        {
          title: `Limits and counterarguments around ${topic}`,
          url: "https://www.rand.org/",
          registrableDomain: "rand.org",
          role: "counterweight",
          selectionReason: "An independent perspective selected to expose limitations and contested assumptions.",
        },
      ],
    };
  }

  const detail = /^\/api\/newsletters\/([^/]+)$/.exec(path);
  if (detail) {
    if (detail[1] === "new-learning-path") {
      demoCreatedNewsletter ??= {
        id: "new-learning-path",
        name: "Evaluating AI agents",
        topic: "How to evaluate AI agents in production",
        learnerLevel: "intermediate",
        learnerGoal: "Design reliable evaluation systems",
        lessonMinutes: 12,
        sourceMode: "discovered",
        sourceReviewMode: "auto",
        sources: [],
        scheduleTime: "08:00",
        timeZone: "UTC",
        active: true,
        emailEnabled: false,
        aiExplorationEnabled: false,
        siteVisible: false,
        issueCount: 1,
        generatedCount: 0,
        sentCount: 0,
      };
      demoCreatedIssue ??= {
        id: "new-learning-path-issue-1",
        status: "queued",
        publicationState: "private",
        createdAt: new Date().toISOString(),
      };
      return {
        newsletter: demoCreatedNewsletter,
        newsletters: [demoCreatedNewsletter, ...newsletters],
        issues: [demoCreatedIssue],
        sourceSummary: { provided: 0, discovered: 0, healthy: 0, needsAttention: 0 },
        sourceCatalog: [],
        resendConfigured: true,
      };
    }
    const newsletter =
      newsletters.find((item) => item.id === detail[1]) ?? newsletters[0];
    return {
      newsletter,
      newsletters,
      issues: issuesByNewsletter[newsletter.id] ?? issuesByNewsletter["ai-evaluation"],
      sourceSummary: {
        provided: newsletter.sources.length,
        discovered: 0,
        healthy: newsletter.sources.length,
        needsAttention: 0,
      },
      sourceCatalog: newsletter.sources.map((source, index) => ({
        id: `${newsletter.id}-source-${index + 1}`,
        displayName: source.name,
        canonicalUrl: source.url,
        health: "healthy",
        origin: "provided",
        kind: "publication",
        state: "active",
      })),
      resendConfigured: true,
    };
  }

  if (path === "/api/newsletters/new-learning-path/delivery" && method === "POST") {
    if (demoCreatedNewsletter) demoCreatedNewsletter.emailEnabled = Boolean(requestBody?.enabled);
    return { enabled: Boolean(requestBody?.enabled) };
  }

  const issue = /^\/api\/issues\/([^/]+)$/.exec(path);
  if (issue && method === "GET") {
    const issueId = decodeURIComponent(issue[1]);
    const newsletter = newsletters.find((item) =>
      (issuesByNewsletter[item.id] ?? []).some((itemIssue) => itemIssue.id === issueId),
    ) ?? newsletters[0];
    const newsletterIssues = issuesByNewsletter[newsletter.id] ?? [];
    const currentIssue = newsletterIssues.find((itemIssue) => itemIssue.id === issueId)
      ?? newsletterIssues[0];
    const currentIndex = newsletterIssues.findIndex((itemIssue) => itemIssue.id === currentIssue.id);
    const dossier = dossierByIssue[currentIssue.id] ?? dossierByIssue["ai-evaluation-issue-1"];
    return {
      issue: currentIssue,
      newsletter,
      newsletters,
      dossier,
      sources: newsletter.sources.slice(0, 3).map((source) => ({
        name: source.name,
        url: source.url,
      })),
      lessonProgress: demoProgress[issueId]
        ? { issueId, ...demoProgress[issueId] }
        : null,
      retrievals: demoRetrievals[issueId] ?? [],
      navigation: {
        previous: newsletterIssues[currentIndex + 1]
          ? {
            issueId: newsletterIssues[currentIndex + 1].id,
            title: newsletterIssues[currentIndex + 1].title,
            createdAt: newsletterIssues[currentIndex + 1].createdAt,
          }
          : null,
        next: currentIndex > 0
          ? {
            issueId: newsletterIssues[currentIndex - 1].id,
            title: newsletterIssues[currentIndex - 1].title,
            createdAt: newsletterIssues[currentIndex - 1].createdAt,
          }
          : null,
        nextReviewAt: null,
      },
    };
  }

  const moderation = /^\/api\/issues\/([^/]+)\/moderation$/.exec(requestURL.pathname);
  if (moderation) {
    const issueId = decodeURIComponent(moderation[1]);
    const current = demoModeration[issueId] ?? {
      state: "clear" as const,
      reason: "",
      corrections: [],
      reports: [],
      actions: [],
    };
    if (method === "POST") {
      const nextState = requestBody?.state === "held" ? "held" : "clear";
      current.state = nextState;
      current.reason = requestBody?.reason ?? "";
      current.actions = [{
        id: `demo-moderation-${current.actions.length + 1}`,
        action: nextState,
        reason: current.reason,
        createdAt: new Date().toISOString(),
      }, ...current.actions];
    }
    demoModeration[issueId] = current;
    return current;
  }

  const corrections = /^\/api\/issues\/([^/]+)\/corrections$/.exec(requestURL.pathname);
  if (corrections && method === "POST") {
    const issueId = decodeURIComponent(corrections[1]);
    const current = demoModeration[issueId] ?? {
      state: "clear" as const,
      reason: "",
      corrections: [],
      reports: [],
      actions: [],
    };
    const correction = {
      id: `demo-correction-${current.corrections.length + 1}`,
      body: requestBody?.body ?? "",
      createdAt: new Date().toISOString(),
    };
    current.corrections = [...current.corrections, correction];
    demoModeration[issueId] = current;
    return correction;
  }

  const progress = /^\/api\/issues\/([^/]+)\/progress$/.exec(requestURL.pathname);
  if (progress && method === "POST") {
    const saved = {
      issueId: progress[1],
      progress: requestBody?.progress ?? 0,
      updatedAt: new Date().toISOString(),
    };
    demoProgress[progress[1]] = saved;
    return saved;
  }

  const retrieval = /^\/api\/issues\/([^/]+)\/retrievals\/([^/]+)$/.exec(requestURL.pathname);
  if (retrieval && method === "POST") {
    const issueId = decodeURIComponent(retrieval[1]);
    const promptKey = decodeURIComponent(retrieval[2]);
    const now = new Date().toISOString();
    const existing = (demoRetrievals[issueId] ?? []).find((item) => item.promptKey === promptKey);
    if (existing) {
      existing.response = requestBody?.skipped ? "" : requestBody?.response ?? existing.response;
      existing.skipped = Boolean(requestBody?.skipped);
      existing.updatedAt = now;
      if (method === "POST") existing.revealedAt = now;
      return existing;
    }
    const saved = {
      issueId,
      promptKey,
      response: requestBody?.skipped ? "" : requestBody?.response ?? "",
      skipped: Boolean(requestBody?.skipped),
      revealedAt: method === "POST" ? now : undefined,
      updatedAt: now,
    };
    demoRetrievals[issueId] = [...(demoRetrievals[issueId] ?? []), saved];
    return saved;
  }

  if (path === "/api/me/site/claim" && method === "POST") {
    return {
      site: {
        ...demoSite,
        username: requestBody?.username ?? demoSite.username,
        displayName: requestBody?.displayName ?? demoSite.displayName,
        visibility: "private",
        searchIndexing: false,
      },
    };
  }

  if (path === "/api/me/site/settings" && method === "POST") {
    const visibility = requestBody?.visibility ?? demoSite.visibility;
    return {
      site: {
        ...demoSite,
        visibility,
        searchIndexing:
          visibility === "public"
            ? requestBody?.searchIndexing ?? demoSite.searchIndexing
            : false,
        displayName: requestBody?.displayName ?? demoSite.displayName,
        description: requestBody?.description ?? demoSite.description,
      },
    };
  }

  if (method === "POST") {
    return { ok: true, site: demoSite };
  }

  throw new Error(`No demo response is configured for ${path}.`);
}
