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

export const demoSite = {
  username: "maya",
  displayName: "Maya’s Learning Garden",
  description: "Notes on cities, intelligence, and the systems quietly shaping everyday life.",
  visibility: "public",
  url: "https://maya.learnloom.blog",
};

export function demoResponse(path, options = {}) {
  const method = (options.method ?? "GET").toUpperCase();

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
      resendConfigured: true,
    };
  }

  if (method === "POST") {
    return { ok: true, site: demoSite };
  }

  throw new Error(`No demo response is configured for ${path}.`);
}

