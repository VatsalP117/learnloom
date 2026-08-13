import { ArrowRight, BookOpen, Check, ExternalLink } from "lucide-react";
import { useEffect } from "react";
import "@fontsource/manrope/latin-400.css";
import "@fontsource/manrope/latin-500.css";
import "@fontsource/manrope/latin-600.css";
import "@fontsource/manrope/latin-700.css";
import BrandMark from "./BrandMark";
import { appOrigin } from "./config";
import "./canonical-dossier.css";

export const canonicalSources = [
  {
    id: 1,
    name: "NIST AI Risk Management Framework Core",
    kind: "Official framework",
    url: "https://airc.nist.gov/airmf-resources/airmf/5-sec-core/",
    use: "Connects measurement to intended use, risk, context, and ongoing monitoring.",
  },
  {
    id: 2,
    name: "OpenAI: Evals drive the next chapter of AI",
    kind: "Practitioner guidance",
    url: "https://openai.com/index/evals-drive-next-chapter-of-ai/",
    use: "Practical guidance for task-specific evals, clear criteria, and continuous iteration.",
  },
  {
    id: 3,
    name: "Judging LLM-as-a-Judge with MT-Bench and Chatbot Arena",
    kind: "Research paper",
    url: "https://arxiv.org/abs/2306.05685",
    use: "Evidence that capable model judges can align with human preferences in studied settings, with known biases.",
  },
  {
    id: 4,
    name: "G-Eval: NLG Evaluation using GPT-4",
    kind: "Research paper",
    url: "https://arxiv.org/abs/2303.16634",
    use: "Shows how explicit criteria and structured evaluation steps can improve model-based judging.",
  },
  {
    id: 5,
    name: "Large Language Models are not Fair Evaluators",
    kind: "Research paper",
    url: "https://arxiv.org/abs/2305.17926",
    use: "Documents order and preference biases that make an automated judge unsafe as an unquestioned oracle.",
  },
] as const;

function Cite({ source }: { source: number }) {
  const item = canonicalSources.find((candidate) => candidate.id === source);
  if (!item) return null;
  return <a className="cd-cite" href={item.url} target="_blank" rel="noreferrer" aria-label={`Source ${source}: ${item.name}`}>{source}</a>;
}

export default function CanonicalDossier() {
  useEffect(() => {
    const previousTitle = document.title;
    document.title = "Can You Trust an AI Research Brief? | Learnloom Dossier";
    const description = "A complete, source-linked Learnloom Dossier on building and calibrating a 12-case evaluator for AI research briefs.";
    let meta = document.querySelector<HTMLMetaElement>('meta[name="description"]');
    const previousDescription = meta?.content;
    if (!meta) {
      meta = document.createElement("meta");
      meta.name = "description";
      document.head.append(meta);
    }
    meta.content = description;
    return () => {
      document.title = previousTitle;
      if (meta && previousDescription !== undefined) meta.content = previousDescription;
    };
  }, []);

  return (
    <div className="cd-page">
      <header className="cd-nav">
        <a className="cd-brand" href="/" aria-label="Learnloom home"><span><BrandMark /></span>Learnloom</a>
        <span className="cd-nav-label">Public Dossier · Complete example</span>
        <a className="cd-nav-cta" href={`${appOrigin}/sign-up`}>Build my path <ArrowRight size={15} /></a>
      </header>

      <main>
        <section className="cd-hero">
          <div className="cd-hero-inner">
            <p className="cd-kicker">AI evaluation · Dossier 01</p>
            <h1>Can you trust an<br /><em>AI research brief?</em></h1>
            <p className="cd-deck">Build a small evaluator that distinguishes a polished answer from a reliable one—and knows when an automated judge needs a human.</p>
            <dl>
              <div><dt>Time</dt><dd>10 minutes</dd></div>
              <div><dt>Outcome</dt><dd>A 12-case evaluation plan</dd></div>
              <div><dt>Sources</dt><dd>5 visible references</dd></div>
            </dl>
          </div>
        </section>

        <div className="cd-layout">
          <article className="cd-lesson">
            <section className="cd-objective">
              <span><BookOpen size={20} /></span>
              <div><p>By the end</p><strong>You can turn “this answer looks good” into a testable release decision with explicit cases, criteria, and escalation rules.</strong></div>
            </section>

            <section id="problem">
              <p className="cd-section-number">01 · The problem</p>
              <h2>Polish is not reliability.</h2>
              <p className="cd-lead">A confident, well-formatted research brief may still cite an obsolete policy, omit a contradiction, or invent the bridge between evidence and conclusion.</p>
              <p>A demo asks whether the system can produce something impressive. An evaluation asks whether it succeeds on representative tasks, fails safely on costly cases, and clears a threshold tied to a real decision. NIST’s framework treats measurement as contextual: validity, reliability, limits, and monitoring all belong to the system’s intended use—not to a score in isolation. <Cite source={1} /></p>
              <aside className="cd-model"><strong>Hold this model</strong><span>Evaluation = cases × criteria × decision rule. A benchmark score without a release consequence is only a measurement.</span></aside>
            </section>

            <section id="cases">
              <p className="cd-section-number">02 · Build the case set</p>
              <h2>Twelve cases are enough to learn—not enough to declare safety.</h2>
              <p>Start with actual work. Write the expected behavior and the expensive failure before choosing a metric. Task-specific evals become more useful when criteria are unambiguous and the set grows from observed failures. <Cite source={2} /></p>
              <div className="cd-case-grid">
                <div><span>3</span><strong>Routine</strong><p>Common requests with clear evidence and expected outputs.</p></div>
                <div><span>3</span><strong>Ambiguous</strong><p>Incomplete, conflicting, or time-sensitive evidence.</p></div>
                <div><span>3</span><strong>High-cost</strong><p>Errors that can trigger bad legal, financial, or operational choices.</p></div>
                <div><span>3</span><strong>Adversarial</strong><p>Instructions that conflict with policy, evidence, or the user’s actual goal.</p></div>
              </div>
              <p className="cd-caption">The mix forces the evaluator to test ordinary usefulness and boundary behavior. It is a prototype set; expand it with every meaningful production failure.</p>
            </section>

            <section id="rubric">
              <p className="cd-section-number">03 · Make success atomic</p>
              <h2>A rubric should make disagreement inspectable.</h2>
              <p>A single 1–10 “quality” score hides why an answer failed. Prefer independent yes/no criteria a reviewer can verify. G-Eval found structured, criteria-led model evaluation more aligned with human judgments than several conventional automated metrics in its studied summarization and dialogue tasks. That is evidence for disciplined judging—not proof that a model judge is ground truth. <Cite source={4} /></p>

              <div className="cd-worked">
                <p className="cd-worked-label">Worked example · Vendor policy brief</p>
                <h3>“What changed in Vendor X’s data-retention policy?”</h3>
                <p>The evidence pack contains an old policy, a current policy, a contradictory FAQ, and one claim with no supporting document.</p>
                <ol>
                  <li><Check size={16} /><span><strong>Grounded:</strong> every material claim is supported by supplied evidence.</span></li>
                  <li><Check size={16} /><span><strong>Current:</strong> the answer identifies the effective date and correct version.</span></li>
                  <li><Check size={16} /><span><strong>Usable:</strong> it follows the requested format and links evidence at the claim.</span></li>
                  <li><Check size={16} /><span><strong>Honest:</strong> unresolved contradictions and missing evidence are flagged.</span></li>
                </ol>
              </div>
            </section>

            <section id="calibration">
              <p className="cd-section-number">04 · Calibrate the judge</p>
              <h2>Automate volume. Keep humans on the boundary.</h2>
              <p>Model judges can show high agreement with human preferences in some settings, but documented biases include position and self-preference effects. <Cite source={3} /> <Cite source={5} /> Calibrate before trusting the automation:</p>
              <ol className="cd-steps">
                <li><span>1</span><p><strong>Blind the comparison.</strong> Remove model names and irrelevant styling.</p></li>
                <li><span>2</span><p><strong>Human-label an audit slice.</strong> Have two people independently score 4 of the 12 cases.</p></li>
                <li><span>3</span><p><strong>Run the model judge.</strong> Require a verdict per criterion plus evidence—not a vibes-based total.</p></li>
                <li><span>4</span><p><strong>Swap A/B order.</strong> If the winner changes, route the case to human review.</p></li>
                <li><span>5</span><p><strong>Inspect disagreement.</strong> Revise unclear criteria or add cases before changing the threshold.</p></li>
              </ol>
              <div className="cd-warning"><strong>Important limitation</strong><p>Order swapping detects one class of pairwise bias. It does not validate factual correctness, eliminate correlated model errors, or establish safety. Human review remains mandatory for high-cost cases and uncertain evidence.</p></div>
            </section>

            <section id="practice">
              <p className="cd-section-number">05 · Retrieval practice</p>
              <h2>Close the page. Test the model in your head.</h2>
              <div className="cd-questions">
                <details><summary><span>01</span>Why is an evaluation a decision instrument rather than a benchmark?</summary><p>Because its cases, criteria, and threshold must connect observed behavior to a concrete action such as release, block, revise, or escalate.</p></details>
                <details><summary><span>02</span>What two safeguards make a model judge more trustworthy?</summary><p>One good pair: independently human-label an audit slice, then test pairwise judgments in both presentation orders. Explicit atomic criteria are another essential safeguard.</p></details>
                <details><summary><span>03</span>When should a case route to a human?</summary><p>When consequences are high, evidence is unresolved, graders disagree, or the verdict changes after an irrelevant transformation such as answer order.</p></details>
              </div>
            </section>

            <section id="apply" className="cd-apply">
              <p className="cd-section-number">06 · Apply it</p>
              <h2>Your next 15 minutes</h2>
              <p>Take a customer-call summarizer. Add one high-cost case where the customer withdraws consent midway through the call. Score two binary criteria: <strong>Does the summary exclude disallowed material?</strong> and <strong>Does it flag the consent transition?</strong> Any failure or grader disagreement routes to a human—never to automatic release.</p>
            </section>
          </article>

          <aside className="cd-aside">
            <div className="cd-aside-card">
              <p>In this Dossier</p>
              <nav><a href="#problem">01 · The problem</a><a href="#cases">02 · Case set</a><a href="#rubric">03 · Rubric</a><a href="#calibration">04 · Calibration</a><a href="#practice">05 · Practice</a><a href="#apply">06 · Apply it</a></nav>
            </div>
            <div className="cd-aside-card cd-sources">
              <p>Source drawer</p>
              {canonicalSources.map((source) => (
                <details key={source.id}>
                  <summary><span>{source.id}</span><strong>{source.name}</strong></summary>
                  <small>{source.kind}</small><p>{source.use}</p>
                  <a href={source.url} target="_blank" rel="noreferrer">Open source <ExternalLink size={12} /></a>
                </details>
              ))}
            </div>
          </aside>
        </div>

        <section className="cd-cta">
          <p>That was one Dossier.</p>
          <h2>What do you want to<br /><em>understand next?</em></h2>
          <p>Give Learnloom a topic. It will find and evaluate a starting evidence base, build the learning path, and help you remember it.</p>
          <a href={`${appOrigin}/sign-up`}>Build my learning path <ArrowRight size={17} /></a>
          <small>You can review the sources before the first lesson.</small>
        </section>
      </main>
    </div>
  );
}
