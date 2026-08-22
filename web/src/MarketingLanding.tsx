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
  X,
} from "lucide-react";
import { useEffect, useState } from "react";
import "@fontsource/manrope/latin-400.css";
import "@fontsource/manrope/latin-500.css";
import "@fontsource/manrope/latin-600.css";
import "@fontsource/manrope/latin-700.css";
import BrandMark from "./BrandMark";
import { appOrigin, personalSiteHost } from "./config";
import "./marketing.css";

export default function MarketingLanding() {
  const [menuOpen, setMenuOpen] = useState(false);

  useEffect(() => {
    document.title = "Learnloom | Stay current. Build understanding that compounds.";
  }, []);

  return (
    <div className="ll-page">
      <header className={`ll-nav${menuOpen ? " menu-open" : ""}`}>
        <a className="ll-brand" href="#top" aria-label="Learnloom home">
          <span className="ll-brand-mark"><BrandMark /></span>
          <span>Learnloom</span>
        </a>
        <nav className="ll-nav-links" id="ll-main-navigation" aria-label="Main navigation">
          <a href="#how-it-works" onClick={() => setMenuOpen(false)}>How it works</a>
          <a href="#why-learnloom" onClick={() => setMenuOpen(false)}>Why Learnloom</a>
          <a href="#pricing" onClick={() => setMenuOpen(false)}>Pricing</a>
          <a href="/guides" onClick={() => setMenuOpen(false)}>Guides</a>
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
            <p className="ll-kicker"><span /> For professionals following fast-moving fields</p>
            <h1>
              Stay current.
              <span className="ll-hero-accent">Build understanding that compounds.</span>
            </h1>
            <p className="ll-hero-description">
              Each day, Learnloom checks credible sources for what is worth
              learning, prepares one focused lesson when the evidence supports
              it, prompts active recall, and makes the next lesson build on what
              you know.
            </p>
            <div className="ll-hero-actions">
              <a className="ll-button ll-button-dark" href={`${appOrigin}/sign-up`}>
                Build my learning path <ArrowRight size={17} />
              </a>
              <a className="ll-text-link" href={`${appOrigin}/examples/ai-evaluation`}>
                Read a complete Dossier <ChevronRight size={16} />
              </a>
            </div>
          </div>

          <div className="ll-hero-proof" aria-label="Learnloom learning path benefits">
            <span><strong>Daily rhythm</strong> led by evidence</span>
            <span><strong>Visible sources</strong> you can inspect</span>
            <span><strong>Active recall</strong> shapes the next lesson</span>
          </div>

          <LearningLoopPreview />
        </section>

        <section className="ll-flow-section" id="how-it-works">
          <div className="ll-flow-heading">
            <p className="ll-eyebrow">A maintained learning path</p>
            <h2>Stop managing the feeds.<br />Start building command.</h2>
          </div>
          <div className="ll-flow-grid">
            <FlowCard
              number="01"
              icon={<Search size={21} />}
              title="Set the learning intent"
              copy="Name the topic, your current level, and the capability you want. Learnloom discovers and validates useful sources—or uses yours."
              visual={<SourceVisual />}
            />
            <FlowCard
              number="02"
              icon={<BrainCircuit size={21} />}
              title="Learn the next useful concept"
              copy="A focused Dossier explains the mechanism, works through an example, challenges weak assumptions, and keeps its evidence visible."
              visual={<ThinkingVisual />}
            />
            <FlowCard
              number="03"
              icon={<BookOpen size={21} />}
              title="Retrieve, then adapt"
              copy="Recall turns reading into a learning cycle. Your response and Learning History shape what deserves your attention next."
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
            <a className="ll-text-link ll-text-link-dark" href={`${appOrigin}/examples/ai-evaluation`}>
              Read a complete Dossier <ArrowRight size={16} />
            </a>
          </div>
          <DossierPreview />
        </section>

        <section className="ll-comparison-section" id="why-learnloom" aria-labelledby="comparison-heading">
          <div className="ll-section-intro">
            <p className="ll-eyebrow">Why Learnloom</p>
            <h2 id="comparison-heading">Information tools help you find.<br />Learnloom helps you progress.</h2>
            <p>Use Learnloom for a topic you need to follow and master over time. It maintains the source–lesson–retrieval loop that other tools leave you to assemble.</p>
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
            <article><strong>Claims stay checkable</strong><p>Material factual claims link to frozen source evidence when the mapping is available. Weak coverage can defer a lesson instead of manufacturing certainty.</p><a href={`${appOrigin}/examples/ai-evaluation`}>Inspect a Dossier <ArrowRight size={13} /></a></article>
            <article><strong>Private by default</strong><p>New paths and lessons begin private or draft. Publishing, search indexing, and follow-by-email each require an explicit action.</p><a href="/privacy">Privacy details <ArrowRight size={13} /></a></article>
            <article><strong>Models have limits</strong><p>Learnloom ranks signals and produces instructional synthesis; it does not certify truth. Corrections and source reporting remain available.</p><a href="/how-learnloom-works">How it works <ArrowRight size={13} /></a></article>
          </div>
        </section>

        <section className="ll-pricing-section" id="pricing" aria-labelledby="pricing-heading">
          <div className="ll-section-intro">
            <p className="ll-eyebrow">Straightforward paid beta pricing</p>
            <h2 id="pricing-heading">Pay for a learning practice,<br />not a pile of generated words.</h2>
            <p>Both plans include the complete source, lesson, retrieval, review, archive, and publishing loop. Choose by how many subjects you want to keep moving—not by an artificial monthly lesson quota.</p>
          </div>
          <div className="ll-pricing-grid">
            <article><p>Essential</p><h3>$9</h3><span>per month</span><ul><li><Check size={15} /> Up to 3 learning streams</li><li><Check size={15} /> Unlimited generated lessons</li><li><Check size={15} /> Private archive, review, and publishing</li></ul><a className="ll-button ll-button-dark" href={`${appOrigin}/sign-up`}>Choose Essential <ArrowRight size={15} /></a></article>
            <article className="featured"><p>Pro</p><h3>$19</h3><span>per month</span><ul><li><Check size={15} /> Unlimited learning streams</li><li><Check size={15} /> Unlimited generated lessons</li><li><Check size={15} /> Full learning and publishing loop</li></ul><a className="ll-button ll-button-dark" href={`${appOrigin}/sign-up`}>Choose Pro <ArrowRight size={15} /></a></article>
          </div>
          <p className="ll-pricing-note">No permanent free plan. Taxes may be added or included based on location and are shown before payment. Paddle acts as merchant of record and provides invoices. Cancel in the hosted portal; existing lessons remain readable, while new generation stops when paid access ends.</p>
        </section>

        <section className="ll-email-section">
          <div className="ll-email-art">
            <div className="ll-mail-card ll-mail-card-back">
              <span>Learnloom</span><i />
            </div>
            <div className="ll-mail-card">
              <div className="ll-mail-top">
                <span className="ll-brand-mark"><BrandMark /></span>
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

        <section className="ll-subdomain-section" id="learning-home">
          <div className="ll-section-intro">
            <p className="ll-eyebrow">A durable home for what you understand</p>
            <h2>Your learning path<br />doesn’t disappear into chat history.</h2>
            <p>
              Every lesson, source, review, and connection returns to your private,
              searchable library. When there is something worth sharing, publish
              only what you choose at an address like <strong>{personalSiteHost("maya")}</strong>.
            </p>
          </div>
          <PersonalHomePreview />
        </section>

        <section className="ll-quote-section">
          <Quote size={31} />
          <blockquote>
            The web keeps producing more to follow.<br />
            Learnloom makes your understanding <em>keep moving.</em>
          </blockquote>
          <p>Designed for professionals who need current knowledge to become usable judgment.</p>
        </section>

        <section className="ll-final-cta">
          <div className="ll-final-clouds" />
          <div className="ll-final-content">
            <span className="ll-brand-mark"><BrandMark /></span>
            <p className="ll-eyebrow">One topic is enough to begin</p>
            <h2>Stop rebuilding context.<br /><em>Start compounding it.</em></h2>
            <p>Choose a paid plan, start a private learning path, and let each focused session build on the last. Publishing stays optional.</p>
            <a className="ll-button ll-button-dark" href={`${appOrigin}/sign-up`}>
              Get started with Learnloom <ArrowRight size={17} />
            </a>
          </div>
        </section>
      </main>

      <footer className="ll-footer">
        <div className="ll-footer-brand">
          <a className="ll-brand" href="#top"><span className="ll-brand-mark"><BrandMark /></span><span>Learnloom</span></a>
          <p>A maintained learning path for fast-moving fields.</p>
        </div>
        <div className="ll-footer-links">
          <div><strong>On this page</strong><a href="#how-it-works">How it works</a><a href="#why-learnloom">Why Learnloom</a><a href="#pricing">Pricing</a></div>
          <div><strong>Learn</strong><a href="/guides">Learning guides</a><a href={`${appOrigin}/examples/ai-evaluation`}>Complete Dossier</a><a href="/editorial-principles">Editorial principles</a></div>
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

function PersonalHomePreview() {
  return (
    <div className="ll-domain-demo" aria-label="Example personal learning home at maya.learnloom.blog">
      <div className="ll-domain-window">
        <div className="ll-browser-bar">
          <span className="ll-browser-dots"><i /><i /><i /></span>
          <div><span className="ll-browser-lock">●</span>{personalSiteHost("maya")}</div>
          <span />
        </div>
        <div className="ll-public-site">
          <div className="ll-public-nav"><span>Maya’s learning home</span><span>Built with Learnloom</span></div>
          <div className="ll-public-hero">
            <p>Following with intention</p>
            <h3>Ideas worth understanding,<br />kept in one place.</h3>
            <span>3 learning paths · 14 Dossiers</span>
          </div>
          <div className="ll-public-cards">
            <article><span>AI evaluation</span><strong>Can you trust an AI research brief?</strong><small>5 sources · 10 min</small></article>
            <article><span>Systems thinking</span><strong>Why feedback loops change outcomes</strong><small>4 sources · 8 min</small></article>
            <article><span>Product craft</span><strong>Make better decisions with less noise</strong><small>6 sources · 12 min</small></article>
          </div>
        </div>
      </div>
      <p className="ll-domain-note"><Check size={14} /> Private by default. Publish a path only when you choose.</p>
    </div>
  );
}

function LearningLoopPreview() {
  return (
    <div className="ll-dashboard-shell" aria-label="Illustrated Learnloom learning loop walkthrough">
      <div className="ll-dashboard-top">
        <div className="ll-mini-brand"><span className="ll-brand-mark"><BrandMark /></span><strong>Learnloom</strong></div>
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
