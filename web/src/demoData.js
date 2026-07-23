const createdAt = (daysAgo, hour = 8) => {
  const value = new Date();
  value.setDate(value.getDate() - daysAgo);
  value.setHours(hour, 12, 0, 0);
  return value.toISOString();
};

const newsletters = [
  {
    id: "urban-systems",
    name: "Urban Systems Field Notes",
    topic: "How hidden infrastructure, ecology, and policy shape the cities we inherit.",
    learnerLevel: "intermediate",
    learnerGoal: "See a city as a living system—and recognize the forces that keep shaping it long after they disappear from view.",
    lessonMinutes: 18,
    sources: [
      { name: "MIT Urban Studies", url: "https://dusp.mit.edu/", limit: 6 },
      { name: "Places Journal", url: "https://placesjournal.org/", limit: 6 },
      { name: "The Nature of Cities", url: "https://www.thenatureofcities.com/", limit: 5 },
    ],
    scheduleTime: "08:00",
    timeZone: "Asia/Kolkata",
    active: true,
    emailEnabled: true,
    emailRecipients: ["maya@example.com"],
    aiExplorationEnabled: true,
    publicSlug: "urban-systems-field-notes",
    siteVisible: true,
    issueCount: 14,
    generatedCount: 14,
    sentCount: 14,
  },
  {
    id: "intelligence",
    name: "Intelligence, Explained",
    topic: "Mental models for understanding modern AI systems without losing sight of their limits.",
    learnerLevel: "advanced",
    learnerGoal: "Build an accurate, durable model of how modern AI systems learn, reason, fail, and affect real institutions.",
    lessonMinutes: 22,
    sources: [
      { name: "Distill", url: "https://distill.pub/", limit: 5 },
      { name: "Anthropic Research", url: "https://www.anthropic.com/research", limit: 5 },
      { name: "Stanford HAI", url: "https://hai.stanford.edu/news", limit: 5 },
    ],
    scheduleTime: "07:30",
    timeZone: "Asia/Kolkata",
    active: true,
    emailEnabled: true,
    emailRecipients: ["maya@example.com"],
    aiExplorationEnabled: false,
    publicSlug: "intelligence-explained",
    siteVisible: true,
    issueCount: 21,
    generatedCount: 20,
    sentCount: 20,
  },
  {
    id: "climate",
    name: "Climate Signals",
    topic: "Reading the evidence behind climate risk, adaptation, and planetary change.",
    learnerLevel: "intermediate",
    learnerGoal: "Separate meaningful climate signals from daily noise and connect global evidence to local decisions.",
    lessonMinutes: 15,
    sources: [
      { name: "Carbon Brief", url: "https://www.carbonbrief.org/", limit: 6 },
      { name: "NASA Climate", url: "https://climate.nasa.gov/", limit: 5 },
      { name: "Our World in Data", url: "https://ourworldindata.org/climate-change", limit: 5 },
    ],
    scheduleTime: "18:00",
    timeZone: "Asia/Kolkata",
    active: false,
    emailEnabled: false,
    emailRecipients: [],
    aiExplorationEnabled: false,
    publicSlug: "climate-signals",
    siteVisible: true,
    issueCount: 12,
    generatedCount: 12,
    sentCount: 8,
  },
];

const issueTitles = [
  "Why cities remember the shape of their rivers",
  "The infrastructure we notice only when it fails",
  "How street grids preserve old systems of power",
  "What urban heat islands reveal about inequality",
  "Why informal transit often outperforms the map",
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
      publicationState: index === 4 ? "hidden" : "published",
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

const dossierByIssue = {
  "urban-systems-issue-1": {
    readTime: 18,
    deck:
      "A river can disappear from the map without disappearing from the city. Trace the old water, and streets, density, and risk start to make more sense.",
    objective:
      "Use buried waterways as a mental model for reading how old infrastructure keeps shaping present-day urban life.",
    sections: [
      {
        label: "The mechanism",
        heading: "Cities keep the shape of what they cover.",
        paragraphs: [
          "When a stream is channelled, filled, or built over, its visible surface disappears first. The low ground, the soil, the drainage path, and the habits of settlement remain. A city inherits those constraints even when its maps tell a cleaner story.",
          "That is why the same pattern appears across very different places: roads bend around old floodplains, dense development gathers on higher ground, and heavy rain exposes the routes water still wants to take. The buried river is not a metaphor. It is a piece of infrastructure with its interface removed.",
        ],
        callout:
          "The useful question is not “Where did the river go?” but “What is the river still making possible—and difficult?”",
      },
      {
        label: "A worked example",
        heading: "Read Bengaluru from the water outward.",
        paragraphs: [
          "Bengaluru’s lakes and interconnected drainage channels were designed as a system, not as isolated blue patches. As the city expanded, roads and buildings interrupted those connections. During intense rain, water follows the older topography, while the newer city experiences that movement as flooding.",
          "This changes the design problem. The solution is not only to move water away faster; it is to restore room for the system to hold, slow, and share water. A map of lakes becomes a map of relationships between ecology, housing, roads, and public decisions.",
        ],
      },
      {
        label: "Skeptical review",
        heading: "The model explains a lot. It does not explain everything.",
        paragraphs: [
          "Old waterways are powerful evidence, but they are not a single-cause theory of urban risk. Drain maintenance, land use, rainfall intensity, construction quality, and unequal access to protection all matter. The same historical pattern can produce very different outcomes depending on who has the power to change it.",
          "Treat the buried-river model as a starting lens. It helps you notice a hidden dependency, then asks you to look for the institutions and incentives that keep the dependency in place.",
        ],
      },
    ],
    retrieval: [
      "Why can a covered stream continue to shape a city’s streets and flood risk?",
      "What changes when a network of lakes is treated as one system?",
      "Which factors would you check before blaming every flood on old waterways?",
    ],
    application:
      "Choose one familiar street that floods after heavy rain. Look for the low points, nearby water bodies, and recent construction. Sketch the path water might be trying to take—and note what your sketch cannot yet explain.",
  },
};

export const demoSite = {
  username: "maya",
  displayName: "Maya’s Learning Garden",
  description: "Notes on cities, intelligence, and the systems quietly shaping everyday life.",
  visibility: "public",
  url: "https://maya.learnloom.blog",
};

export function demoResponse(path, options = {}) {
  const method = (options.method ?? "GET").toUpperCase();

  if (path === "/api/workspace" && method === "GET") {
    const issues = newsletters.flatMap((newsletter) =>
      (issuesByNewsletter[newsletter.id] ?? []).map((issue) => ({
        ...issue,
        newsletter,
      })),
    );
    const reviews = issues
      .filter((issue) => issue.status === "generated")
      .slice(0, 8)
      .map((issue) => {
        const dossier = dossierByIssue[issue.id] ?? dossierByIssue["urban-systems-issue-1"];
        return {
          issueId: issue.id,
          objective: dossier.objective,
          questions: dossier.retrieval,
        };
      });
    return {
      summary: {
        newsletters: newsletters.length,
        active: newsletters.filter((item) => item.active).length,
        generated: newsletters.reduce((total, item) => total + item.generatedCount, 0),
      },
      newsletters,
      issues,
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
    return { newsletter: newsletters[0] };
  }

  const detail = /^\/api\/newsletters\/([^/]+)$/.exec(path);
  if (detail) {
    const newsletter =
      newsletters.find((item) => item.id === detail[1]) ?? newsletters[0];
    return {
      newsletter,
      newsletters,
      issues: issuesByNewsletter[newsletter.id] ?? issuesByNewsletter["urban-systems"],
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

  const issue = /^\/api\/issues\/([^/]+)$/.exec(path);
  if (issue && method === "GET") {
    const issueId = decodeURIComponent(issue[1]);
    const newsletter = newsletters.find((item) =>
      (issuesByNewsletter[item.id] ?? []).some((itemIssue) => itemIssue.id === issueId),
    ) ?? newsletters[0];
    const newsletterIssues = issuesByNewsletter[newsletter.id] ?? [];
    const currentIssue = newsletterIssues.find((itemIssue) => itemIssue.id === issueId)
      ?? newsletterIssues[0];
    const dossier = dossierByIssue[currentIssue.id] ?? dossierByIssue["urban-systems-issue-1"];
    return {
      issue: currentIssue,
      newsletter,
      newsletters,
      dossier,
      sources: newsletter.sources.slice(0, 3).map((source) => ({
        name: source.name,
        url: source.url,
      })),
    };
  }

  if (path === "/api/me/site/claim" && method === "POST") {
    return {
      site: {
        ...demoSite,
        username: options.body?.username ?? demoSite.username,
        displayName: options.body?.displayName ?? demoSite.displayName,
        visibility: "private",
      },
    };
  }

  if (path === "/api/me/site/settings" && method === "POST") {
    return {
      site: {
        ...demoSite,
        visibility: options.body?.visibility ?? demoSite.visibility,
        displayName: options.body?.displayName ?? demoSite.displayName,
        description: options.body?.description ?? demoSite.description,
      },
    };
  }

  if (method === "POST") {
    return { ok: true, site: demoSite };
  }

  throw new Error(`No demo response is configured for ${path}.`);
}
