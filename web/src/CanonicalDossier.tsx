import { ArrowRight } from "lucide-react";
import { useEffect } from "react";
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

// The full Dossier arc. The example below renders these sections in exactly
// this order; the test suite asserts the order-sensitive contract.
export const canonicalDossierOrder = [
  "learning-objective",
  "two-minute-recall",
  "why-this-matters",
  "mental-model",
  "how-it-works",
  "worked-example",
  "common-misconception",
  "practical-experiment",
  "takeaway",
  "retrieval-practice",
  "application-challenge",
  "answer-key",
  "ai-exploration",
  "sources",
] as const;

function Cite({ source }: { source: number }) {
  const item = canonicalSources.find((candidate) => candidate.id === source);
  if (!item) return null;
  return (
    <a
      className="cd-cite"
      href={item.url}
      target="_blank"
      rel="noreferrer"
      aria-label={`Source ${source}: ${item.name}`}
    >
      [{source}]
    </a>
  );
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
        <p className="cd-kicker">AI evaluation · Dossier 01</p>
        <h1>Can you trust an AI research brief?</h1>
        <p className="cd-deck">Build a small evaluator that distinguishes a polished answer from a reliable one—and knows when an automated judge needs a human.</p>
        <hr />

        <article>
          <section id="learning-objective">
            <h3>Learning objective</h3>
            <p>By the end of this Dossier you can turn “this answer looks good” into a testable release decision: explicit cases, independent criteria, a threshold tied to a real decision, and an escalation rule that keeps a human on the boundary.</p>
          </section>

          <section id="two-minute-recall">
            <h3>Two-minute recall</h3>
            <p>A research brief is trustworthy when four things hold: every material claim is grounded in supplied evidence, the evidence is current, the answer is usable, and unresolved contradictions are flagged. An evaluation is a decision instrument—cases × criteria × decision rule—not a score. A benchmark number without a release consequence is only a measurement.</p>
          </section>

          <section id="why-this-matters">
            <h3>Why this matters</h3>
            <p>A confident, well-formatted research brief may still cite an obsolete policy, omit a contradiction, or invent the bridge between evidence and conclusion. Polish is not reliability. A demo asks whether the system can produce something impressive; an evaluation asks whether it succeeds on representative tasks, fails safely on costly cases, and clears a threshold tied to a real decision. NIST’s AI Risk Management Framework treats measurement as contextual: validity, reliability, limits, and ongoing monitoring belong to the system’s intended use and risk context, not to a score in isolation. <Cite source={1} /></p>
            <p>That framing turns evaluation from a grade into governance. The same system can be trustworthy for low-stakes summaries and untrustworthy for high-stakes ones, and only a decision-linked evaluation can tell the two apart.</p>
          </section>

          <section id="mental-model">
            <h3>Mental model</h3>
            <p>Hold this model: <strong>evaluation = cases × criteria × decision rule</strong>.</p>
            <ul>
              <li><strong>Cases</strong> are representative slices of the work the system will actually face—including the failures you fear most.</li>
              <li><strong>Criteria</strong> are independent yes/no checks a reviewer can verify, not a vibes-based total.</li>
              <li><strong>Decision rule</strong> says what happens on failure: revise, block, escalate, or route to a human.</li>
            </ul>
            <p>Every score in the system must be traceable to one of these three parts. Anything else is decoration.</p>
          </section>

          <section id="how-it-works">
            <h3>How it works</h3>
            <p>Start from actual work, not from a template. Write the expected behavior and the expensive failure before choosing a metric. Task-specific evals become more useful when criteria are unambiguous and the case set grows from observed failures. <Cite source={2} /></p>
            <p>A useful prototype set for a research-brief assistant has twelve cases: three routine (common requests with clear evidence and expected outputs), three ambiguous (incomplete, conflicting, or time-sensitive evidence), three high-cost (errors that can trigger bad legal, financial, or operational choices), and three adversarial (instructions that conflict with policy, evidence, or the user’s actual goal). The mix forces the evaluator to test ordinary usefulness and boundary behavior at once. Treat it as a prototype set and expand it with every meaningful production failure.</p>
            <p>Then make success atomic. A single 1–10 “quality” score hides why an answer failed; prefer independent yes/no criteria a reviewer can verify. G-Eval found criteria-led, structured model evaluation aligned more closely with human judgments than several conventional automated metrics in its studied summarization and dialogue tasks—evidence for disciplined judging, not proof that a model judge is ground truth. <Cite source={4} /></p>
            <p>Finally, calibrate the judge before trusting the automation. Model judges can show high agreement with human preferences in some studied settings, but documented biases include position and self-preference effects. <Cite source={3} /> <Cite source={5} /></p>
          </section>

          <section id="worked-example">
            <h3>Worked example · Vendor policy brief</h3>
            <p>Prompt: “What changed in Vendor X’s data-retention policy?” The evidence pack contains an old policy, a current policy, a contradictory FAQ, and one claim with no supporting document. Score four binary criteria:</p>
            <ol>
              <li><strong>Grounded:</strong> every material claim is supported by supplied evidence.</li>
              <li><strong>Current:</strong> the answer identifies the effective date and correct version.</li>
              <li><strong>Usable:</strong> it follows the requested format and links evidence at the claim.</li>
              <li><strong>Honest:</strong> unresolved contradictions and missing evidence are flagged.</li>
            </ol>
            <p>Decision rule: a pass requires all four. Any failure means revise or escalate; anything that requires interpreting conflicting policy routes to a human.</p>
          </section>

          <section id="common-misconception">
            <h3>Common misconception</h3>
            <p>“A model judge that agrees with humans can replace human review.” Agreement on a studied benchmark set does not transfer automatically to your cases, and documented order and preference biases can flip pairwise verdicts. <Cite source={3} /> <Cite source={5} /> The safe practice is structural: keep a human-labeled audit slice, test verdicts in both presentation orders, and treat the model judge as a triage tool that routes disagreement to humans—never as an oracle.</p>
          </section>

          <section id="practical-experiment">
            <h3>Practical experiment · Calibrate a 12-case judge</h3>
            <p>This fifteen-minute experiment fits inside the prototype set. Take a customer-call summarizer. Add one high-cost case where a customer withdraws consent midway through the call. Score two binary criteria: <strong>does the summary exclude disallowed material?</strong> and <strong>does it flag the consent transition?</strong></p>
            <p>Then run the calibration loop over the whole set:</p>
            <ol>
              <li><strong>Blind the comparison.</strong> Remove model names and irrelevant styling.</li>
              <li><strong>Human-label an audit slice.</strong> Have two people independently score four of the twelve cases.</li>
              <li><strong>Run the model judge.</strong> Require a verdict per criterion plus evidence—not a vibes-based total.</li>
              <li><strong>Swap A/B order.</strong> If the winner changes, route the case to human review.</li>
              <li><strong>Inspect disagreement.</strong> Revise unclear criteria or add cases before changing any threshold.</li>
            </ol>
            <p>Any failure or grader disagreement routes to a human—never to automatic release.</p>
          </section>

          <section id="takeaway">
            <h3>Takeaway</h3>
            <p>Automate volume, keep humans on the boundary. Release only when the criteria are atomic, the audit slice agrees, order swaps do not flip verdicts, and high-cost cases have a human-reviewed lane. Two habits protect you: write the expensive failure down before writing the test, and treat every unexplained judge disagreement as a bug in your criteria, not a nuisance.</p>
          </section>

          <section id="retrieval-practice">
            <h3>Retrieval practice</h3>
            <p>Close the page. Test the model in your head.</p>
            <details>
              <summary>Why is an evaluation a decision instrument rather than a benchmark?</summary>
              <p>Because its cases, criteria, and threshold connect observed behavior to a concrete action such as release, block, revise, or escalate. A benchmark score without a release consequence is only a measurement.</p>
            </details>
            <details>
              <summary>What two safeguards make a model judge more trustworthy?</summary>
              <p>One good pair: independently human-label an audit slice, then test pairwise judgments in both presentation orders. Explicit atomic criteria are an essential third safeguard—they make disagreement inspectable.</p>
            </details>
            <details>
              <summary>When should a case route to a human?</summary>
              <p>When consequences are high, evidence is unresolved, graders disagree, or the verdict changes after an irrelevant transformation such as answer order.</p>
            </details>
            <details>
              <summary>What distinguishes a demo from an evaluation?</summary>
              <p>A demo shows the system can produce something impressive. An evaluation tests representative success, safe failure, and a threshold tied to a real decision.</p>
            </details>
          </section>

          <section id="application-challenge">
            <h3>Application challenge</h3>
            <p>Within one working session: pick a summarizer or brief-generator you rely on, define its expensive failure in one sentence, add one high-cost case to its evaluation set, and score two binary criteria on it. Run the order-swap step. If any verdict flips, document it and route that case to human review before the next release.</p>
          </section>

          <section id="answer-key">
            <h3>Answer key</h3>
            <ol className="cd-answers">
              <li><strong>Decision instrument, not benchmark:</strong> cases, criteria, and a threshold tie observed behavior to a concrete action—release, block, revise, or escalate. Without a consequence, a score is only a measurement.</li>
              <li><strong>Two safeguards:</strong> an independently human-labeled audit slice, plus pairwise judgments tested in both presentation orders. Atomic criteria make disagreement inspectable.</li>
              <li><strong>Route to a human when:</strong> consequences are high, evidence is unresolved, graders disagree, or a verdict flips under an irrelevant transformation such as answer order.</li>
              <li><strong>Demo vs. evaluation:</strong> a demo shows capability; an evaluation tests representative success, safe failure, and a decision-linked threshold.</li>
            </ol>
          </section>
        </article>

        <hr />

        <section id="ai-exploration">
          <h3>AI Exploration · Opt-in</h3>
          <p className="cd-label">Speculative — extends beyond the cited sources. These analogies, deductions, scenarios, and suggestions are written for thinking, not for citation.</p>
          <h4>Novel analogy · A kitchen thermometer</h4>
          <p>A thermometer checked only at body temperature can drift badly at roasting temperatures. Calibrate at the temperature you actually cook at: judge agreement measured on one task distribution tells you little about another. That is a reason to audit slices that resemble deployment—not a claim about any specific model.</p>
          <h4>Cross-domain deduction · Financial audit</h4>
          <p>Auditors do not rely on one reviewer’s opinion. They test controls, sample transactions, and require two-person review of material findings. The evaluation analogue is the audit slice (independent double labeling) and the criteria checklist (the control test). The discipline transfers; no numeric result is claimed here.</p>
          <h4>Hypothetical scenario · Order-sensitive briefs</h4>
          <p>Imagine a research brief that reaches opposite conclusions depending on which of two conflicting policy documents is presented first. If your judge flips with it, the case is unstable and a human must decide. Order-swap testing is cheap insurance against exactly this failure mode.</p>
          <h4>Experiment idea · Flip-rate check</h4>
          <p>On your twelve cases, run the judge twice with the evidence pack in reversed order and record how many verdicts flip. A stable judge should flip none. If your run shows flips, treat those cases as human-review triggers. Record your own numbers—none are claimed here.</p>
          <h4>Uncertainty labels</h4>
          <p>Instead of a single score, emit labels: “Grounding: 3 of 4 claims supported, 1 unsupported — escalate.” Labels make disagreement inspectable and give the decision rule something to branch on.</p>
        </section>

        <hr />

        <section id="sources">
          <h2>Sources</h2>
          <ol>
            {canonicalSources.map((source) => (
              <li key={source.id}>
                <strong>{source.name}</strong> — {source.kind}. {source.use}{" "}
                <a href={source.url} target="_blank" rel="noreferrer">{source.url}</a>
              </li>
            ))}
          </ol>
          <p className="cd-label">This Dossier is source-bounded: every empirical claim in the lesson sections cites one of the five sources above, and the worked numbers describe a prototype plan, not measured results. The AI Exploration section is explicitly speculative and did not inform the lesson claims.</p>
          <p className="cd-label">Model output can be wrong. Verify important claims at the linked sources before acting on them.</p>
        </section>
      </main>

      <footer className="cd-cta">
        <h2>Build a path like this</h2>
        <p>Give Learnloom a topic. It will find and evaluate a starting evidence base, build the learning path, and help you remember it.</p>
        <a href={`${appOrigin}/sign-up`}>Build my learning path <ArrowRight size={15} /></a>
        <small>You can review the sources before the first lesson.</small>
      </footer>
    </div>
  );
}
