export const STREAM_TEMPLATE_CATALOG_VERSION = 3;
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
  editorialStatus: "founder_review_pending" | "human_reviewed";
  sources: Array<{ name: string; url: string; limit: number }>;
  sample: {
    objective: string;
    concepts: string[];
    retrievalPrompt: string;
  };
}

// Template IDs are permanent. Material changes create a new per-template
// version; source-health fixes may update URLs without changing the outcome.
// These candidates must not be described as human-reviewed until a founder or
// qualified editor records that review in the catalog.
export const streamTemplates: StreamTemplate[] = [
  {
    id: "ai-evaluation-governance",
    version: 1,
    name: "Evaluate an AI system",
    outcome: "Design a practical evaluation plan connecting risks, metrics, test cases, and release decisions.",
    topic: "Trustworthy AI evaluation, testing, and risk management",
    learnerGoal: "Turn ‘is this model good?’ into a measurable, decision-relevant evaluation brief.",
    learnerLevel: "intermediate",
    lessonMinutes: 12,
    editorialStatus: "founder_review_pending",
    sources: [
      { name: "NIST AI Risk Management Framework", url: "https://www.nist.gov/itl/ai-risk-management-framework", limit: 8 },
      { name: "NIST Generative AI Profile", url: "https://www.nist.gov/publications/artificial-intelligence-risk-management-framework-generative-artificial-intelligence", limit: 8 },
      { name: "NIST AI Resource Center", url: "https://airc.nist.gov/", limit: 8 },
      { name: "NIST AIRC Technical Reports", url: "https://airc.nist.gov/technical-reports/", limit: 8 },
    ],
    sample: {
      objective: "Treat an evaluation as a decision instrument, not a benchmark leaderboard.",
      concepts: ["intended use", "representative cases", "release thresholds"],
      retrievalPrompt: "Name one capability metric, one safety metric, and the decision each should inform.",
    },
  },
  {
    id: "production-rag-quality",
    version: 1,
    name: "Build reliable RAG",
    outcome: "Design and evaluate a retrieval pipeline that returns relevant evidence and produces grounded answers.",
    topic: "Production retrieval-augmented generation, search quality, grounding, and evaluation",
    learnerGoal: "Diagnose whether a weak answer came from retrieval, context assembly, or generation—and choose a measurable fix.",
    learnerLevel: "intermediate",
    lessonMinutes: 12,
    editorialStatus: "founder_review_pending",
    sources: [
      { name: "OpenAI Retrieval guide", url: "https://platform.openai.com/docs/guides/retrieval", limit: 8 },
      { name: "Azure AI Search: RAG overview", url: "https://learn.microsoft.com/en-us/azure/search/retrieval-augmented-generation-overview", limit: 8 },
      { name: "Retrieval-Augmented Generation paper", url: "https://arxiv.org/abs/2005.11401", limit: 8 },
      { name: "OpenAI Evals guide", url: "https://platform.openai.com/docs/guides/evals", limit: 8 },
    ],
    sample: {
      objective: "Separate retrieval quality from answer quality before tuning the system.",
      concepts: ["retrieval relevance", "groundedness", "failure attribution"],
      retrievalPrompt: "A grounded answer omits the key fact: name one retrieval metric, one generation metric, and the next diagnostic test.",
    },
  },
  {
    id: "reliable-ai-agents",
    version: 1,
    name: "Engineer reliable AI agents",
    outcome: "Choose a simple agent architecture with bounded tools, explicit control points, and task-level evaluations.",
    topic: "AI agent workflows, tool use, orchestration, control, and reliability evaluation",
    learnerGoal: "Decide when an agent is justified, constrain what it may do, and test the complete trajectory rather than a final answer alone.",
    learnerLevel: "intermediate",
    lessonMinutes: 12,
    editorialStatus: "founder_review_pending",
    sources: [
      { name: "Anthropic: Building effective agents", url: "https://www.anthropic.com/engineering/building-effective-agents", limit: 8 },
      { name: "OpenAI Agents guide", url: "https://platform.openai.com/docs/guides/agents", limit: 8 },
      { name: "Model Context Protocol architecture", url: "https://modelcontextprotocol.io/docs/learn/architecture", limit: 8 },
      { name: "ReAct paper", url: "https://arxiv.org/abs/2210.03629", limit: 8 },
    ],
    sample: {
      objective: "Start with the least autonomous architecture that can complete the job.",
      concepts: ["workflow vs agent", "tool contracts", "trajectory evaluation"],
      retrievalPrompt: "For a support-refund task, choose a workflow or agent and name one approval boundary and one end-to-end eval.",
    },
  },
  {
    id: "llm-application-security",
    version: 1,
    name: "Threat-model an LLM app",
    outcome: "Create a practical threat model for an LLM application and map its highest-risk paths to layered controls and tests.",
    topic: "LLM application security, prompt injection, excessive agency, data exposure, and supply-chain risk",
    learnerGoal: "Move from a generic safety checklist to concrete trust boundaries, abuse cases, mitigations, and verification evidence.",
    learnerLevel: "intermediate",
    lessonMinutes: 15,
    editorialStatus: "founder_review_pending",
    sources: [
      { name: "OWASP Top 10 for LLM Applications", url: "https://genai.owasp.org/llm-top-10/", limit: 8 },
      { name: "NIST Generative AI Profile", url: "https://www.nist.gov/publications/artificial-intelligence-risk-management-framework-generative-artificial-intelligence", limit: 8 },
      { name: "MITRE ATLAS", url: "https://atlas.mitre.org/", limit: 8 },
      { name: "Google Secure AI Framework", url: "https://saif.google/", limit: 8 },
    ],
    sample: {
      objective: "Model the attacker, trust boundary, and impact before selecting a control.",
      concepts: ["prompt injection", "least privilege", "defense in depth"],
      retrievalPrompt: "For an email-reading agent, trace one indirect prompt-injection path and name preventive, detective, and recovery controls.",
    },
  },
  {
    id: "context-engineering",
    version: 1,
    name: "Engineer model context",
    outcome: "Design a context pipeline that preserves the right instructions and evidence while controlling distraction, staleness, and token cost.",
    topic: "Context engineering for LLM applications and agents, including selection, compression, memory, and prompt structure",
    learnerGoal: "Treat context as a runtime system to measure and curate, rather than a prompt that only grows longer.",
    learnerLevel: "intermediate",
    lessonMinutes: 12,
    editorialStatus: "founder_review_pending",
    sources: [
      { name: "Anthropic: Effective context engineering for AI agents", url: "https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents", limit: 8 },
      { name: "OpenAI Prompt engineering guide", url: "https://platform.openai.com/docs/guides/prompt-engineering", limit: 8 },
      { name: "Lost in the Middle paper", url: "https://arxiv.org/abs/2307.03172", limit: 8 },
      { name: "Model Context Protocol architecture", url: "https://modelcontextprotocol.io/docs/learn/architecture", limit: 8 },
    ],
    sample: {
      objective: "Make every token earn its place in the model's working context.",
      concepts: ["context selection", "position effects", "compaction"],
      retrievalPrompt: "A long-running agent misses an early constraint: propose one selection, one placement, and one compaction fix, with a metric for each.",
    },
  },
  {
    id: "production-ai-observability",
    version: 1,
    name: "Operate AI inference",
    outcome: "Define service-level indicators and diagnose the quality, latency, throughput, and cost tradeoffs of a production AI workload.",
    topic: "Production AI inference performance, observability, batching, caching, serving, and cost control",
    learnerGoal: "Connect traces and service metrics to user-visible quality and choose optimizations without hiding regressions.",
    learnerLevel: "intermediate",
    lessonMinutes: 12,
    editorialStatus: "founder_review_pending",
    sources: [
      { name: "OpenTelemetry GenAI semantic conventions", url: "https://opentelemetry.io/docs/specs/semconv/gen-ai/", limit: 8 },
      { name: "OpenAI Latency optimization guide", url: "https://platform.openai.com/docs/guides/latency-optimization", limit: 8 },
      { name: "vLLM paper", url: "https://arxiv.org/abs/2309.06180", limit: 8 },
      { name: "NVIDIA TensorRT-LLM documentation", url: "https://nvidia.github.io/TensorRT-LLM/", limit: 8 },
    ],
    sample: {
      objective: "Optimize the user-visible service, not an isolated tokens-per-second benchmark.",
      concepts: ["time to first token", "throughput", "quality-cost frontier"],
      retrievalPrompt: "A chat endpoint has fast generation but slow first tokens: name likely phases to trace, one safe optimization, and its guardrail metric.",
    },
  },
];
