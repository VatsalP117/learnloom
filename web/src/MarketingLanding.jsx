import {
  ArrowRight,
  BookOpen,
  BrainCircuit,
  Check,
  ChevronRight,
  Clock3,
  Globe2,
  Mail,
  Menu,
  Quote,
  Search,
  Sparkles,
} from "lucide-react";
import { useEffect } from "react";
import { appOrigin, personalSiteHost } from "./config.js";
import "./marketing.css";

export default function MarketingLanding() {
  useEffect(() => {
    document.title = "Learnloom · A learning home that grows with you";
  }, []);

  return (
    <div className="ll-page">
      <header className="ll-nav">
        <a className="ll-brand" href="#top" aria-label="Learnloom home">
          <BrandMark />
          <span>Learnloom</span>
        </a>
        <nav className="ll-nav-links" aria-label="Main navigation">
          <a href="#product">Product</a>
          <a href="#how-it-works">How it works</a>
          <a href="#dossiers">Dossiers</a>
        </nav>
        <div className="ll-nav-actions">
          <a className="ll-sign-in" href={`${appOrigin}/sign-in`}>Sign in</a>
          <a className="ll-button ll-button-dark ll-button-small" href={`${appOrigin}/sign-up`}>
            Start learning <ArrowRight size={15} />
          </a>
          <button className="ll-menu" type="button" aria-label="Open navigation">
            <Menu size={21} />
          </button>
        </div>
      </header>

      <main id="top">
        <section className="ll-hero">
          <div className="ll-hero-scrim" />
          <div className="ll-hero-copy">
            <p className="ll-kicker"><span /> Your place for deeper learning</p>
            <h1>
              A learning home that
              <br />
              <em>grows with you.</em>
            </h1>
            <p className="ll-hero-description">
              Learnloom turns the sources you trust into thoughtful daily
              Dossiers—then publishes them to a personal corner of the web.
            </p>
            <div className="ll-hero-actions">
              <a className="ll-button ll-button-dark" href={`${appOrigin}/sign-up`}>
                Claim your learning home <ArrowRight size={17} />
              </a>
              <a className="ll-text-link" href="#product">
                See how it works <ChevronRight size={16} />
              </a>
            </div>
          </div>

          <div className="ll-address-pill" aria-label="Example personal Learnloom address">
            <span className="ll-address-status"><Globe2 size={15} /></span>
            <span>{personalSiteHost("maya")}</span>
            <span className="ll-live-dot"><i /> Public</span>
          </div>

          <DashboardPreview />
        </section>

        <section className="ll-subdomain-section" id="product">
          <div className="ll-section-intro">
            <p className="ll-eyebrow">A home, not another inbox</p>
            <h2>Your learning deserves<br />its own address.</h2>
            <p>
              Every Dossier lives in a beautiful, lasting library on your
              personal subdomain. Share it when you want. Keep it private when
              you don’t.
            </p>
          </div>
          <div className="ll-domain-demo">
            <div className="ll-domain-window">
              <div className="ll-browser-bar">
                <span className="ll-browser-dots"><i /><i /><i /></span>
                <div><Globe2 size={13} />{personalSiteHost("maya")}</div>
                <span />
              </div>
              <div className="ll-public-site">
                <div className="ll-public-nav">
                  <span>Maya’s Learning Garden</span>
                  <span>Topics &nbsp; Archive &nbsp; About</span>
                </div>
                <div className="ll-public-hero">
                  <p>Today’s Dossier · 8 min read</p>
                  <h3>Why cities remember<br />the shape of their rivers</h3>
                  <span>Urban Systems · July 19</span>
                </div>
                <div className="ll-public-cards">
                  <article><span>01</span><strong>The mechanism</strong><p>How buried waterways continue to shape streets, density, and risk.</p></article>
                  <article><span>02</span><strong>A worked example</strong><p>Tracing the old streams underneath modern Bengaluru.</p></article>
                  <article><span>03</span><strong>Test your model</strong><p>Three retrieval questions and one field observation.</p></article>
                </div>
              </div>
            </div>
            <div className="ll-domain-note ll-domain-note-one">
              <Check size={15} /> Your own searchable archive
            </div>
            <div className="ll-domain-note ll-domain-note-two">
              <Check size={15} /> Public or private, your choice
            </div>
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
              title="Choose what matters"
              copy="Pick a topic and the sources you trust. Learnloom keeps the signal and leaves the noise."
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
              <li><Check size={17} /> Sources stay attached to every claim</li>
              <li><Check size={17} /> New ideas connect to your Learning History</li>
              <li><Check size={17} /> Questions turn reading into recall</li>
            </ul>
            <a className="ll-text-link ll-text-link-dark" href={`${appOrigin}/sign-up`}>
              Create your first Dossier <ArrowRight size={16} />
            </a>
          </div>
          <DossierPreview />
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
              <p>Urban Systems · Issue 14</p>
              <h3>Why cities remember the shape of their rivers</h3>
              <div className="ll-mail-lines"><i /><i /><i /><i /></div>
              <a>Continue reading on {personalSiteHost("maya")} <ArrowRight size={14} /></a>
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
            <p>Claim your personal Learnloom address and publish your first Dossier.</p>
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
          <div><strong>Product</strong><a href="#product">Personal sites</a><a href="#dossiers">Dossiers</a><a href="#how-it-works">How it works</a></div>
          <div><strong>Account</strong><a href={`${appOrigin}/sign-in`}>Sign in</a><a href={`${appOrigin}/sign-up`}>Get started</a></div>
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

function DashboardPreview() {
  return (
    <div className="ll-dashboard-shell" aria-label="Learnloom dashboard preview">
      <div className="ll-dashboard-top">
        <div className="ll-mini-brand"><BrandMark /><strong>Learnloom</strong></div>
        <div className="ll-dashboard-search"><Search size={12} /> Search your learning <kbd>⌘ K</kbd></div>
        <span className="ll-avatar">MP</span>
      </div>
      <div className="ll-dashboard-body">
        <aside>
          <span className="ll-side-label">Workspace</span>
          <a className="active"><BookOpen size={13} /> Dossiers</a>
          <a><Clock3 size={13} /> Learning history</a>
          <span className="ll-side-label ll-side-spacer">Active topics</span>
          <a><span className="ll-topic-dot blue" /> Urban systems</a>
          <a><span className="ll-topic-dot green" /> Behavioral science</a>
          <a><span className="ll-topic-dot gold" /> Climate technology</a>
        </aside>
        <div className="ll-dashboard-main">
          <div className="ll-dash-heading">
            <div><span>Your intelligence workspace</span><h3>Knowledge Dossiers</h3></div>
            <button><span>+</span> New topic</button>
          </div>
          <div className="ll-dash-cards">
            <MiniDossier icon={<Globe2 size={17} />} title="Urban systems" color="blue" progress="82%" />
            <MiniDossier icon={<BrainCircuit size={17} />} title="Behavioral science" color="green" progress="64%" />
            <MiniDossier icon={<Sparkles size={17} />} title="Climate technology" color="gold" progress="91%" />
          </div>
        </div>
      </div>
    </div>
  );
}

function MiniDossier({ icon, title, color, progress }) {
  return (
    <article className="ll-mini-dossier">
      <div><span className={`ll-mini-icon ${color}`}>{icon}</span><span className="ll-active"><i /> Active</span></div>
      <h4>{title}</h4>
      <p>A guided daily path through sources, mechanisms, and practice.</p>
      <div className="ll-mini-meta"><span>Next issue</span><strong>Tomorrow, 8:00 AM</strong></div>
      <div className="ll-mini-progress"><span><i style={{ width: progress }} /></span><strong>{progress}</strong></div>
      <a>Open Dossier <ChevronRight size={12} /></a>
    </article>
  );
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
      <span className="ll-source-card one"><i>aeon</i><strong>How ideas take root</strong><small>Essay · 12 min</small></span>
      <span className="ll-source-card two"><i>Nature</i><strong>A new view of memory</strong><small>Research · 8 min</small></span>
      <span className="ll-source-card three"><i>MIT</i><strong>Systems that learn</strong><small>Journal · 6 min</small></span>
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
      <div className="ll-archive-row"><i className="blue" /><span><strong>The city beneath the city</strong><small>Urban systems · Jul 18</small></span></div>
      <div className="ll-archive-row"><i className="green" /><span><strong>Why habits resist intention</strong><small>Behavioral science · Jul 17</small></span></div>
      <div className="ll-archive-row"><i className="gold" /><span><strong>Storing the summer sun</strong><small>Climate technology · Jul 16</small></span></div>
    </div>
  );
}

function DossierPreview() {
  return (
    <div className="ll-dossier-preview">
      <div className="ll-dossier-paper">
        <div className="ll-paper-meta"><span>LEARNLOOM DOSSIER</span><span>ISSUE 14 · 8 MIN</span></div>
        <p className="ll-paper-topic">URBAN SYSTEMS</p>
        <h3>Why cities remember the<br />shape of their rivers</h3>
        <p className="ll-paper-deck">A mechanism-focused guide to the waterways hidden beneath modern streets.</p>
        <div className="ll-paper-rule" />
        <p className="ll-paper-label">THE CENTRAL MECHANISM</p>
        <p className="ll-paper-body"><span className="ll-dropcap">A</span> river does not disappear when it is covered. Its valley, soil, drainage, and floodplain continue to govern what can be built above it.</p>
        <div className="ll-paper-callout"><strong>Hold this model</strong><span>Infrastructure can hide a system without replacing its behavior.</span></div>
        <p className="ll-paper-label">RETRIEVAL PRACTICE</p>
        <div className="ll-question"><span>01</span><p>Why can a buried river still influence flooding?</p></div>
        <div className="ll-question"><span>02</span><p>What would you look for on a city map?</p></div>
      </div>
      <div className="ll-source-tab"><span>4</span> cited sources</div>
    </div>
  );
}
