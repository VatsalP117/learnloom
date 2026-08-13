import {
  ArrowRight,
  BookOpen,
  BrainCircuit,
  Check,
  ChevronRight,
  Clock3,
  Mail,
  Menu,
  Quote,
  Search,
  Sparkles,
  X,
} from "lucide-react";
import { useEffect, useState } from "react";
import "@fontsource/manrope/latin-400.css";
import "@fontsource/manrope/latin-500.css";
import "@fontsource/manrope/latin-600.css";
import "@fontsource/manrope/latin-700.css";
import { appOrigin, personalSiteHost } from "./config";
import "./marketing.css";

export default function MarketingLanding() {
  const [menuOpen, setMenuOpen] = useState(false);

  useEffect(() => {
    document.title = "Learnloom | Give us a topic. We’ll build the learning path.";
  }, []);

  return (
    <div className="ll-page">
      <header className={`ll-nav${menuOpen ? " menu-open" : ""}`}>
        <a className="ll-brand" href="#top" aria-label="Learnloom home">
          <BrandMark />
          <span>Learnloom</span>
        </a>
        <nav className="ll-nav-links" id="ll-main-navigation" aria-label="Main navigation">
          <a href="/solutions" onClick={() => setMenuOpen(false)}>Solutions</a>
          <a href="/product/ai-learning-assistant" onClick={() => setMenuOpen(false)}>Product</a>
          <a href="/guides" onClick={() => setMenuOpen(false)}>Guides</a>
          <a href="/examples" onClick={() => setMenuOpen(false)}>Examples</a>
          <a href="#how-it-works" onClick={() => setMenuOpen(false)}>How it works</a>
        </nav>
        <div className="ll-nav-actions">
          <a className="ll-sign-in" href={`${appOrigin}/sign-in`}>Sign in</a>
          <a className="ll-button ll-button-dark ll-button-small" href={`${appOrigin}/sign-up`}>
            Start learning <ArrowRight size={15} />
          </a>
          <button
            className="ll-menu"
            type="button"
            aria-label={menuOpen ? "Close navigation" : "Open navigation"}
            aria-controls="ll-main-navigation"
            aria-expanded={menuOpen}
            onClick={() => setMenuOpen((open) => !open)}
          >
            {menuOpen ? <X size={21} /> : <Menu size={21} />}
          </button>
        </div>
      </header>

      <main id="top">
        <section className="ll-hero">
          <div className="ll-hero-scrim" />
          <div className="ll-hero-copy">
            <p className="ll-kicker"><span /> For AI and software professionals in fast-moving fields</p>
            <h1>
              Give us a topic.
              <br />
              <em>We’ll build the learning path.</em>
            </h1>
            <p className="ll-hero-description">
              Stop rebuilding context from feeds, bookmarks, and one-off chats.
              Learnloom finds and ranks useful sources, teaches the next concept,
              and revisits it before it fades.
            </p>
            <div className="ll-hero-actions">
              <a className="ll-button ll-button-dark" href={`${appOrigin}/sign-up`}>
                Build my learning path <ArrowRight size={17} />
              </a>
              <a className="ll-text-link" href="/examples/ai-evaluation">
                Read a complete Dossier <ChevronRight size={16} />
              </a>
            </div>
          </div>

          <div className="ll-address-pill" aria-label="Complete product example available before signup">
            <span className="ll-address-status"><BookOpen size={15} /></span>
            <span>Complete lesson available before signup</span>
            <span className="ll-live-dot"><i /> Real example</span>
          </div>

          <LearningLoopPreview />
        </section>

        <section className="ll-subdomain-section" id="product">
          <div className="ll-section-intro">
            <p className="ll-eyebrow">One topic becomes a maintained learning system</p>
            <h2>See the whole loop,<br />not another AI answer.</h2>
            <p>
              The path begins privately. You can inspect the provisional source
              portfolio, learn through a short evidence-linked session, attempt
              retrieval, and see why the next lesson changes.
            </p>
          </div>
          <div className="ll-loop-proof" aria-label="Learnloom product sequence">
            <ProofStep number="01" label="Topic" title="Calibrate an evaluator for AI research briefs" detail="Outcome: make defensible release decisions" />
            <ProofStep number="02" label="Source portfolio" title="Official guidance, evaluation research, practitioner evidence" detail="Ranked by role, relevance, authority signals, and coverage" />
            <ProofStep number="03" label="Learning session" title="Build a 12-case evaluator" detail="Mechanism, worked case, limitations, citations · about 12 min" />
            <ProofStep number="04" label="Retrieval" title="Decide when disagreement requires human review" detail="Answer before reveal; rate your recall" />
            <ProofStep number="05" label="Adaptation" title="Next: test order sensitivity and judge calibration" detail="Chosen from gaps, prior concepts, recall, and available time" />
            <a className="ll-button ll-button-dark" href="/examples/ai-evaluation">Inspect the complete lesson <ArrowRight size={16} /></a>
          </div>
        </section>

        <section className="ll-flow-section" id="how-it-works">
          <div className="ll-flow-heading">
            <p className="ll-eyebrow">Quietly working in the background</p>
            <h2>From today’s sources<br />to tomorrow’s understanding.</h2>
          </div>
          <div className="ll-flow-grid">
            <FlowCard
              number="01"
              icon={<Search size={21} />}
              title="Name what you want to understand"
              copy="Start with a topic. Learnloom discovers and validates useful sources—or you can bring your own."
              visual={<SourceVisual />}
            />
            <FlowCard
              number="02"
              icon={<BrainCircuit size={21} />}
              title="Build real understanding"
              copy="Every Dossier connects ideas, explains mechanisms, checks misconceptions, and cites its sources."
              visual={<ThinkingVisual />}
            />
            <FlowCard
              number="03"
              icon={<BookOpen size={21} />}
              title="Return to what you know"
              copy="Your Learning History creates continuity, so each new lesson builds on the last."
              visual={<ArchiveVisual />}
            />
          </div>
        </section>

        <section className="ll-dossier-section" id="dossiers">
          <div className="ll-dossier-copy">
            <p className="ll-eyebrow">More than a summary</p>
            <h2>Built to become<br /><em>understanding.</em></h2>
            <p>
              Learnloom doesn’t hand you a pile of links. It creates a
              source-grounded lesson with a clear mechanism, worked example,
              skeptical review, and retrieval practice.
            </p>
            <ul>
              <li><Check size={17} /> Sources remain visible for verification</li>
              <li><Check size={17} /> New ideas connect to your Learning History</li>
              <li><Check size={17} /> Questions turn reading into recall</li>
            </ul>
            <a className="ll-text-link ll-text-link-dark" href="/examples/ai-evaluation">
              Read a complete Dossier <ArrowRight size={16} />
            </a>
          </div>
          <DossierPreview />
        </section>

        <section className="ll-comparison-section" aria-labelledby="comparison-heading">
          <div className="ll-section-intro">
            <p className="ll-eyebrow">Choose the workflow, not the category label</p>
            <h2 id="comparison-heading">Different tools solve<br />different parts of learning.</h2>
            <p>Learnloom is for a topic you need to follow and master over time. These comparisons describe workflow defaults, not universal limitations.</p>
          </div>
          <div className="ll-comparison-table" role="table" aria-label="Learning workflow comparison">
            <ComparisonRow tool="General chat assistants" strength="Fast explanations and questions" tradeoff="You repeatedly supply context, judge sources, and decide what comes next" />
            <ComparisonRow tool="NotebookLM" strength="Explore and synthesize a source collection" tradeoff="You assemble the collection and maintain the learning sequence" />
            <ComparisonRow tool="Readwise / read-later" strength="Capture, resurface, and annotate material" tradeoff="The reading queue is not automatically turned into a source-ranked curriculum" />
            <ComparisonRow tool="Recall / knowledge maps" strength="Summaries and connected personal knowledge" tradeoff="Learnloom centers a scheduled lesson–retrieval–adaptation loop" />
            <ComparisonRow tool="Learnloom" strength="Topic-to-source portfolio-to-adaptive lesson loop" tradeoff="A focused learning system, not a general research workspace or notes replacement" emphasized />
          </div>
        </section>

        <section className="ll-trust-section" aria-labelledby="trust-heading">
          <div><p className="ll-eyebrow">Trust is a product surface</p><h2 id="trust-heading">Inspect the evidence.<br />Keep the boundary.</h2></div>
          <div className="ll-trust-grid">
            <article><strong>Explainable selection</strong><p>See source roles and why candidates were chosen. Prefer, block, replace, or provide your own sources.</p><a href="/editorial-principles">Source methodology <ArrowRight size={13} /></a></article>
            <article><strong>Claims stay checkable</strong><p>Material factual claims link to frozen source evidence. Weak coverage can defer a lesson instead of manufacturing certainty.</p><a href="/examples/ai-evaluation">Inspect a Dossier <ArrowRight size={13} /></a></article>
            <article><strong>Private by default</strong><p>New paths and lessons begin private or draft. Publishing, search indexing, and follow-by-email each require an explicit action.</p><a href="/privacy">Privacy details <ArrowRight size={13} /></a></article>
            <article><strong>Models have limits</strong><p>Learnloom ranks signals and produces instructional synthesis; it does not certify truth. Corrections and source reporting remain available.</p><a href="/how-learnloom-works">How it works <ArrowRight size={13} /></a></article>
          </div>
        </section>

        <section className="ll-pricing-section" id="pricing" aria-labelledby="pricing-heading">
          <div className="ll-section-intro">
            <p className="ll-eyebrow">Design-partner pricing</p>
            <h2 id="pricing-heading">Experience the loop free.<br />Pay when it becomes a habit.</h2>
            <p>This is our launch hypothesis, not an “unlimited AI” promise. We will keep the allowance visible and revise it only from measured cost and learning outcomes.</p>
          </div>
          <div className="ll-pricing-grid">
            <article><p>Free</p><h3>$0</h3><span>per month</span><ul><li><Check size={15} /> One starter learning path</li><li><Check size={15} /> 3 lesson generations per 30 days</li><li><Check size={15} /> Private archive and review</li></ul><a className="ll-button ll-button-dark" href={`${appOrigin}/sign-up`}>Start free <ArrowRight size={15} /></a></article>
            <article className="featured"><p>Pro design partner</p><h3>$15</h3><span>per month · price hypothesis</span><ul><li><Check size={15} /> 30 lesson generations per 30 days</li><li><Check size={15} /> Full learning and publishing loop</li><li><Check size={15} /> Cancel in the hosted billing portal</li></ul><a className="ll-button ll-button-dark" href={`${appOrigin}/sign-up`}>Join the design-partner beta <ArrowRight size={15} /></a></article>
          </div>
          <p className="ll-pricing-note">Taxes may be added or included based on your location and will be shown before payment. Paddle acts as merchant of record and provides invoices. Existing lessons remain readable after cancellation; new generation returns to the Free allowance.</p>
        </section>

        <section className="ll-email-section">
          <div className="ll-email-art">
            <div className="ll-mail-card ll-mail-card-back">
              <span>Learnloom</span><i />
            </div>
            <div className="ll-mail-card">
              <div className="ll-mail-top">
                <BrandMark />
                <span>Today’s Dossier</span>
                <span>8:00 AM</span>
              </div>
              <p>AI Evaluation · Dossier 01</p>
              <h3>Can you trust an AI research brief?</h3>
              <div className="ll-mail-lines"><i /><i /><i /><i /></div>
              <a>Continue reading on {personalSiteHost("alex")} <ArrowRight size={14} /></a>
            </div>
          </div>
          <div className="ll-email-copy">
            <span className="ll-round-icon"><Mail size={21} /></span>
            <p className="ll-eyebrow">Email, when you want it</p>
            <h2>Your Dossier can meet<br />you in your inbox, too.</h2>
            <p>
              Email is a gentle nudge, not the destination. Open the full
              Dossier on your learning home, where every issue stays organized,
              searchable, and yours.
            </p>
          </div>
        </section>

        <section className="ll-quote-section">
          <Quote size={31} />
          <blockquote>
            The web gives us endless things to read.<br />
            Learnloom gives each idea <em>somewhere to live.</em>
          </blockquote>
          <p>Designed for curious people building a lifelong body of knowledge.</p>
        </section>

        <section className="ll-final-cta">
          <div className="ll-final-clouds" />
          <div className="ll-final-content">
            <BrandMark />
            <p className="ll-eyebrow">Your learning home is waiting</p>
            <h2>Make curiosity<br /><em>a place you return to.</em></h2>
            <p>Start a private path from one topic. Publishing stays optional.</p>
            <a className="ll-button ll-button-dark" href={`${appOrigin}/sign-up`}>
              Get started with Learnloom <ArrowRight size={17} />
            </a>
          </div>
        </section>
      </main>

      <footer className="ll-footer">
        <div className="ll-footer-brand">
          <a className="ll-brand" href="#top"><BrandMark /><span>Learnloom</span></a>
          <p>Current sources, woven into durable understanding.</p>
        </div>
        <div className="ll-footer-links">
          <div><strong>Product</strong><a href="/product/ai-learning-assistant">AI learning assistant</a><a href="/product/trusted-source-learning">Trusted-source learning</a><a href="/how-learnloom-works">How Learnloom works</a></div>
          <div><strong>Learn</strong><a href="/guides">Learning guides</a><a href="/examples">Public examples</a><a href="/editorial-principles">Editorial principles</a></div>
          <div><strong>Account</strong><a href={`${appOrigin}/sign-in`}>Sign in</a><a href={`${appOrigin}/sign-up`}>Get started</a></div>
          <div><strong>Legal</strong><a href="/privacy">Privacy</a><a href="/terms">Terms</a></div>
        </div>
        <div className="ll-footer-bottom">
          <span>© 2026 Learnloom</span>
          <span>Built for durable understanding.</span>
        </div>
      </footer>
    </div>
  );
}

function BrandMark() {
  return <span className="ll-brand-mark"><Sparkles size={16} strokeWidth={2.2} /></span>;
}

function LearningLoopPreview() {
  return (
    <div className="ll-dashboard-shell" aria-label="Illustrated Learnloom learning loop walkthrough">
      <div className="ll-dashboard-top">
        <div className="ll-mini-brand"><BrandMark /><strong>Learnloom</strong></div>
        <div className="ll-dashboard-search"><Search size={12} /> AI evaluator reliability</div>
        <span className="ll-avatar">01</span>
      </div>
      <div className="ll-dashboard-body">
        <aside>
          <span className="ll-side-label">Workspace</span>
          <a className="active"><Search size={13} /> Source portfolio</a>
          <a><BookOpen size={13} /> Focused lesson</a>
          <a><BrainCircuit size={13} /> Retrieval</a>
          <a><Clock3 size={13} /> Adapted next step</a>
        </aside>
        <div className="ll-dashboard-main">
          <div className="ll-dash-heading">
            <div><span>Illustrated product walkthrough</span><h3>From topic to the next useful concept</h3></div>
            <button>Private by default</button>
          </div>
          <div className="ll-dash-cards">
            <MiniDossier icon={<Search size={17} />} title="Evidence portfolio" color="blue" capability="Roles and reasons visible" />
            <MiniDossier icon={<BookOpen size={17} />} title="12-minute session" color="green" capability="Claims linked to sources" />
            <MiniDossier icon={<BrainCircuit size={17} />} title="Adaptive return" color="gold" capability="Recall updates what comes next" />
          </div>
        </div>
      </div>
    </div>
  );
}

function MiniDossier({ icon, title, color, capability }) {
  return (
    <article className="ll-mini-dossier">
      <div><span className={`ll-mini-icon ${color}`}>{icon}</span><span className="ll-active"><i /> Active</span></div>
      <h4>{title}</h4>
      <p>One stage in a connected learning cycle—not an invented activity feed.</p>
      <div className="ll-mini-meta"><span>Product behavior</span><strong>Inspectable and durable</strong></div>
      <div className="ll-mini-capability"><Check size={11} /><strong>{capability}</strong></div>
      <a>Open Dossier <ChevronRight size={12} /></a>
    </article>
  );
}

function ProofStep({ number, label, title, detail }) {
  return <article><span>{number}</span><div><small>{label}</small><strong>{title}</strong><p>{detail}</p></div></article>;
}

function ComparisonRow({ tool, strength, tradeoff, emphasized = false }) {
  return <div className={emphasized ? "emphasized" : ""} role="row"><strong role="cell">{tool}</strong><span role="cell">{strength}</span><p role="cell">{tradeoff}</p></div>;
}

function FlowCard({ number, icon, title, copy, visual }) {
  return (
    <article className="ll-flow-card">
      <div className="ll-flow-visual">{visual}</div>
      <div className="ll-flow-card-copy">
        <div><span>{icon}</span><i>{number}</i></div>
        <h3>{title}</h3>
        <p>{copy}</p>
      </div>
    </article>
  );
}

function SourceVisual() {
  return (
    <div className="ll-source-visual">
      <span className="ll-source-card one"><i>NIST</i><strong>AI RMF Core</strong><small>Official framework</small></span>
      <span className="ll-source-card two"><i>OpenAI</i><strong>Evals in practice</strong><small>Practitioner guidance</small></span>
      <span className="ll-source-card three"><i>Research</i><strong>LLM judge bias</strong><small>Peer-reviewed evidence</small></span>
    </div>
  );
}

function ThinkingVisual() {
  return (
    <div className="ll-thinking-visual">
      <span className="ll-orbit orbit-one" />
      <span className="ll-orbit orbit-two" />
      <span className="ll-thinking-core"><BrainCircuit size={26} /></span>
      <span className="ll-thinking-chip chip-one">Mechanism</span>
      <span className="ll-thinking-chip chip-two">Evidence</span>
      <span className="ll-thinking-chip chip-three">Practice</span>
    </div>
  );
}

function ArchiveVisual() {
  return (
    <div className="ll-archive-visual">
      <div className="ll-archive-top"><span>Learning history</span><Search size={13} /></div>
      <div className="ll-archive-row"><i className="blue" /><span><strong>Can you trust an AI brief?</strong><small>AI evaluation · Lesson 1</small></span></div>
      <div className="ll-archive-row"><i className="green" /><span><strong>Calibrate a model judge</strong><small>AI evaluation · Lesson 2</small></span></div>
      <div className="ll-archive-row"><i className="gold" /><span><strong>Test order sensitivity</strong><small>AI evaluation · Next</small></span></div>
    </div>
  );
}

function DossierPreview() {
  return (
    <div className="ll-dossier-preview">
      <div className="ll-dossier-paper">
        <div className="ll-paper-meta"><span>LEARNLOOM DOSSIER</span><span>ISSUE 14 · 8 MIN</span></div>
        <p className="ll-paper-topic">AI EVALUATION</p>
        <h3>Can you trust an<br />AI research brief?</h3>
        <p className="ll-paper-deck">Build a 12-case evaluator that knows when an automated judge needs a human.</p>
        <div className="ll-paper-rule" />
        <p className="ll-paper-label">THE CENTRAL MECHANISM</p>
        <p className="ll-paper-body"><span className="ll-dropcap">A</span>n evaluation connects representative cases and explicit criteria to a real release decision. Polish alone cannot establish reliability.</p>
        <div className="ll-paper-callout"><strong>Hold this model</strong><span>Evaluation = cases × criteria × decision rule.</span></div>
        <p className="ll-paper-label">RETRIEVAL PRACTICE</p>
        <div className="ll-question"><span>01</span><p>Why is a benchmark not yet a release decision?</p></div>
        <div className="ll-question"><span>02</span><p>When must a case route to a human?</p></div>
      </div>
      <div className="ll-source-tab"><span>4</span> cited sources</div>
    </div>
  );
}
