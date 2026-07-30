export const STREAM_TEMPLATE_CATALOG_VERSION = 1;
export const STREAM_TEMPLATE_OWNER = "Learnloom product";

export interface StreamTemplate {
  id: string;
  version: number;
  name: string;
  outcome: string;
  topic: string;
  learnerGoal: string;
  learnerLevel: "beginner" | "intermediate" | "advanced";
  lessonMinutes: number;
  sources: Array<{ name: string; url: string; limit: number }>;
  sample: {
    objective: string;
    concepts: string[];
    retrievalPrompt: string;
  };
}

// Template IDs are permanent. Material changes create a new per-template
// version; source-health fixes may update URLs without changing the outcome.
export const streamTemplates: StreamTemplate[] = [
  {
    id: "ai-systems-evidence",
    version: 1,
    name: "Understand AI systems",
    outcome: "Explain how current AI methods work, where evidence is strong, and how they fail.",
    topic: "How modern AI systems learn, reason, and fail in practice",
    learnerGoal: "Build an evidence-led mental model I can use to evaluate new AI research and products.",
    learnerLevel: "intermediate",
    lessonMinutes: 20,
    sources: [
      { name: "arXiv Artificial Intelligence", url: "https://rss.arxiv.org/rss/cs.AI", limit: 8 },
      { name: "Nature Machine Learning", url: "https://www.nature.com/subjects/machine-learning.rss", limit: 8 },
    ],
    sample: {
      objective: "Distinguish model capability from reliable task performance.",
      concepts: ["evaluation validity", "distribution shift", "calibration"],
      retrievalPrompt: "What evidence would change your confidence in an AI capability claim?",
    },
  },
  {
    id: "climate-systems",
    version: 1,
    name: "Read climate change clearly",
    outcome: "Connect climate mechanisms, observed changes, and practical adaptation without losing uncertainty.",
    topic: "Climate mechanisms, evidence, impacts, and adaptation",
    learnerGoal: "Explain important climate findings accurately and connect them to real adaptation decisions.",
    learnerLevel: "intermediate",
    lessonMinutes: 20,
    sources: [
      { name: "Nature Climate Change", url: "https://www.nature.com/subjects/climate-change.rss", limit: 8 },
      { name: "NASA Earth Observatory", url: "https://science.nasa.gov/feed/earth-observatory/natural-events/", limit: 8 },
    ],
    sample: {
      objective: "Connect a measured climate signal to its mechanism and practical consequence.",
      concepts: ["forcing", "attribution", "adaptation"],
      retrievalPrompt: "How would you separate a climate trend from one extreme event?",
    },
  },
  {
    id: "energy-transition",
    version: 1,
    name: "Follow the energy transition",
    outcome: "Reason about technologies, economics, and constraints shaping clean-energy deployment.",
    topic: "The technology and economics of the clean-energy transition",
    learnerGoal: "Compare energy pathways using mechanisms, system constraints, and credible evidence.",
    learnerLevel: "intermediate",
    lessonMinutes: 20,
    sources: [
      { name: "Nature Energy", url: "https://www.nature.com/subjects/energy.rss", limit: 8 },
      { name: "NASA Technology", url: "https://www.nasa.gov/feed/", limit: 8 },
    ],
    sample: {
      objective: "Compare an energy technology by cost, system value, and deployment constraint.",
      concepts: ["capacity factor", "grid flexibility", "learning curves"],
      retrievalPrompt: "Why can the cheapest generator still be costly for the whole system?",
    },
  },
];
