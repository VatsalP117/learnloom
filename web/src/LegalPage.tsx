import { ArrowLeft, ArrowRight, Mail } from "lucide-react";
import { useEffect, type ReactNode } from "react";
import "@fontsource/manrope/latin-400.css";
import "@fontsource/manrope/latin-500.css";
import "@fontsource/manrope/latin-600.css";
import "@fontsource/manrope/latin-700.css";
import "@fontsource/bricolage-grotesque/latin-500.css";
import "@fontsource/bricolage-grotesque/latin-600.css";
import BrandMark from "./BrandMark";
import { appOrigin } from "./config";
import "./legal.css";

const EFFECTIVE_DATE = "August 12, 2026";

export default function LegalPage() {
  const isTerms = window.location.pathname === "/terms";

  useEffect(() => {
    document.title = `${isTerms ? "Terms of Service" : "Privacy Policy"} · Learnloom`;
  }, [isTerms]);

  return (
    <div className="legal-page">
      <header className="legal-nav">
        <a className="legal-brand" href="/" aria-label="Learnloom home">
          <span><BrandMark /></span>
          <strong>Learnloom</strong>
        </a>
        <nav aria-label="Legal navigation">
          <a className={!isTerms ? "current" : ""} href="/privacy">Privacy</a>
          <a className={isTerms ? "current" : ""} href="/terms">Terms</a>
          <a className="legal-sign-in" href={`${appOrigin}/sign-in`}>
            Sign in <ArrowRight size={14} />
          </a>
        </nav>
      </header>

      <main>
        <a className="legal-back" href="/">
          <ArrowLeft size={15} /> Back to Learnloom
        </a>
        {isTerms ? <TermsContent /> : <PrivacyContent />}
      </main>

      <footer className="legal-footer">
        <div>
          <span><BrandMark /></span>
          <strong>Learnloom</strong>
        </div>
        <p>Understanding, built one lesson at a time.</p>
        <a href="mailto:support@learnloom.blog">
          <Mail size={14} /> support@learnloom.blog
        </a>
      </footer>
    </div>
  );
}

function PrivacyContent() {
  return (
    <article className="legal-document">
      <header className="legal-heading">
        <p>Trust and privacy</p>
        <h1>Privacy Policy</h1>
        <span>Effective {EFFECTIVE_DATE}</span>
        <p>
          Learnloom is a personal learning service. This policy explains what
          information we handle, why we need it, and the choices you have.
        </p>
      </header>

      <LegalSection title="1. Information we collect">
        <p>We collect information in the following categories:</p>
        <ul>
          <li><strong>Account and opt-in information:</strong> your name, email address, authentication identifiers, and account status supplied through Clerk; or an email address you submit to follow a public learning path.</li>
          <li><strong>Learning information:</strong> topics, questions, source URLs, learning preferences, schedules, lesson progress, retrieval responses, and publishing choices.</li>
          <li><strong>Content:</strong> generated Dossiers, personal-site information, and messages you send to support.</li>
          <li><strong>Technical information:</strong> IP address, browser and device information, request logs, security events, and service-performance data.</li>
          <li><strong>Delivery information:</strong> email-delivery status and related diagnostics when you choose email delivery.</li>
        </ul>
      </LegalSection>

      <LegalSection title="2. How we use information">
        <p>We use this information to provide and personalize Learnloom, create source-grounded lessons, maintain your Learning History, deliver requested emails, publish content you deliberately make public, protect the service, respond to support requests, and meet legal obligations.</p>
        <p>We do not sell personal information, and Learnloom does not use your learning activity for third-party advertising.</p>
      </LegalSection>

      <LegalSection title="3. AI processing and source retrieval">
        <p>Learnloom sends relevant learning instructions, source material, and prior-learning context to configured AI model providers to generate Dossiers. Source URLs may also be requested from third-party websites or search services. Do not submit confidential information that you do not want processed for these purposes.</p>
        <p>Learnloom may retrieve publicly accessible pages when a site does not publish a robots.txt file. We do not bypass authentication, paywalls, CAPTCHA challenges, or other technical access controls. Public availability does not give Learnloom ownership of source material or permission to republish it without limit.</p>
        <p>Generated material can be incomplete or incorrect. You should review important claims and follow the attached sources.</p>
      </LegalSection>

      <LegalSection title="4. Service providers">
        <p>We use trusted service providers to operate Learnloom, including Clerk for identity, hosting and database providers, object storage, AI model providers, source-discovery services, Resend for transactional, learning, and confirmed public-path email delivery, and Paddle as merchant of record for payments, subscription management, invoices, tax calculation and remittance, and fraud prevention. They process information under their own contractual and security obligations.</p>
      </LegalSection>

      <LegalSection title="5. Public publishing">
        <p>Your account and learning streams are private by default. If you make your personal site, a stream, or a Dossier public, the selected content can be viewed by anyone and may be indexed, copied, or shared outside Learnloom. You can change future visibility, but copies already made by others may remain.</p>
      </LegalSection>

      <LegalSection title="6. Retention and deletion">
        <p>We retain account information and learning content while your account is active. After a verified deletion, Learnloom disables access immediately, removes stored lesson files, and erases active database learning content. We retain a one-way identity tombstone to prevent stale identity events from recreating the account and a minimal erasure receipt for up to 400 days. Provider backups, security logs, or records required by law may persist for their limited retention periods before expiring.</p>
      </LegalSection>

      <LegalSection title="7. Security">
        <p>We use administrative, technical, and organizational safeguards designed to protect your information, including authenticated access controls, encrypted transport, signed webhooks, and restricted service credentials. No online service can guarantee absolute security.</p>
      </LegalSection>

      <LegalSection title="8. Your choices and rights">
        <p>You can update learning and publishing settings in Learnloom, stop email delivery, make eligible content private, or request access, correction, export, or deletion of personal information. Rights vary by location, and we may need to verify your identity before fulfilling a request.</p>
      </LegalSection>

      <LegalSection title="9. Children">
        <p>Learnloom is not directed to children under 13, or under the minimum age required by local law. We do not knowingly collect personal information from children below that age.</p>
      </LegalSection>

      <LegalSection title="10. International processing">
        <p>Learnloom and its providers may process information in countries other than your own. Where required, we use appropriate safeguards for international transfers.</p>
      </LegalSection>

      <LegalSection title="11. Changes and contact">
        <p>We may update this policy as Learnloom evolves. We will revise the effective date and provide additional notice when a material change requires it.</p>
        <p>Questions or privacy requests can be sent to <a href="mailto:support@learnloom.blog">support@learnloom.blog</a>.</p>
      </LegalSection>
    </article>
  );
}

function TermsContent() {
  return (
    <article className="legal-document">
      <header className="legal-heading">
        <p>Using Learnloom</p>
        <h1>Terms of Service</h1>
        <span>Effective {EFFECTIVE_DATE}</span>
        <p>
          These terms govern your access to Learnloom. By creating an account
          or using the service, you agree to them.
        </p>
      </header>

      <LegalSection title="1. Eligibility and accounts">
        <p>You must be at least 13 years old and legally able to enter into these terms. If local law requires a higher age, that higher age applies. You are responsible for accurate account information, safeguarding access to your account, and activity performed through it.</p>
      </LegalSection>

      <LegalSection title="2. The service">
        <p>Learnloom retrieves sources you select or discover, uses automated systems to produce learning material, maintains a learning archive, and can deliver or publish content according to your settings. Features may change as the service develops.</p>
      </LegalSection>

      <LegalSection title="3. Acceptable use">
        <p>You may not use Learnloom to violate law or another person’s rights; access accounts or systems without authorization; distribute malware or harmful code; interfere with service operation; evade limits or security controls; generate unlawful, abusive, or deceptive material; or submit content you do not have the right to use.</p>
      </LegalSection>

      <LegalSection title="4. Your content and instructions">
        <p>You retain your rights in content and instructions you submit. You give Learnloom a limited permission to host, copy, process, retrieve, transform, deliver, and publish that material only as needed to operate the service and honor your settings.</p>
        <p>You are responsible for the sources you select, your right to use submitted material, and the decision to publish generated content.</p>
      </LegalSection>

      <LegalSection title="5. Generated content">
        <p>AI-generated Dossiers may contain errors, omissions, or outdated information. They are learning aids, not professional medical, legal, financial, or other regulated advice. Review important claims, consult the cited sources, and use qualified professionals when appropriate.</p>
      </LegalSection>

      <LegalSection title="6. Third-party services and sources">
        <p>Learnloom relies on third-party identity, hosting, email, search, storage, source, and AI services. External websites and source material are controlled by their respective owners and may have separate terms. We are not responsible for third-party content or availability.</p>
        <p>Learnloom may retrieve publicly accessible source pages, including pages on sites that do not publish a robots.txt file, to produce attributed instructional synthesis. We do not bypass authentication, paywalls, CAPTCHA challenges, or other technical access controls, and retrieval does not grant us ownership or unrestricted republication rights. Source owners and rights holders can request review, correction, or removal at <a href="mailto:support@learnloom.blog">support@learnloom.blog</a>.</p>
      </LegalSection>

      <LegalSection title="7. Learnloom property">
        <p>Learnloom’s software, branding, interface, and service design are protected by applicable intellectual-property laws. These terms do not grant permission to copy, resell, reverse engineer, or create competing services from protected parts of Learnloom except where law expressly allows it.</p>
      </LegalSection>

      <LegalSection title="8. Suspension and termination">
        <p>You may stop using Learnloom or request account deletion at any time. We may restrict or terminate access when reasonably necessary to protect users or the service, respond to legal requirements, address material violations, or discontinue the service. Where practical, we will provide notice.</p>
      </LegalSection>

      <LegalSection title="9. Plans, billing, taxes, and refunds">
        <p>Free and paid plans have the generation allowances and billing periods shown before purchase. Paid subscriptions renew automatically until canceled. Paddle acts as merchant of record: the checkout shows the applicable price, currency, and taxes before you pay, and Paddle provides invoices and payment support through its hosted portal.</p>
        <p>You can cancel through the billing portal. Cancellation normally takes effect at the end of the paid period; existing learning content remains readable and future generation returns to the applicable Free allowance. Payment failure may create a limited grace period before new generation pauses. Refunds are handled according to applicable law and the policy presented at checkout; contact <a href="mailto:support@learnloom.blog">support@learnloom.blog</a> if a purchase is incorrect or the service was not delivered as described.</p>
      </LegalSection>

      <LegalSection title="10. Service availability and disclaimers">
        <p>Learnloom is provided on an “as available” basis. To the extent permitted by law, we disclaim implied warranties of merchantability, fitness for a particular purpose, non-infringement, and uninterrupted or error-free operation. Nothing in these terms excludes rights that cannot legally be excluded.</p>
      </LegalSection>

      <LegalSection title="11. Limitation of liability">
        <p>To the extent permitted by law, Learnloom and its operators will not be liable for indirect, incidental, special, consequential, or punitive damages, or for lost data, profits, opportunities, or goodwill arising from use of the service. Any liability that cannot be excluded will be limited to the minimum amount permitted by law.</p>
      </LegalSection>

      <LegalSection title="12. Changes">
        <p>We may update these terms as Learnloom changes. We will revise the effective date and provide additional notice when required. Continued use after an update takes effect means you accept the revised terms.</p>
      </LegalSection>

      <LegalSection title="13. Contact">
        <p>Questions about these terms can be sent to <a href="mailto:support@learnloom.blog">support@learnloom.blog</a>.</p>
      </LegalSection>
    </article>
  );
}

function LegalSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section>
      <h2>{title}</h2>
      {children}
    </section>
  );
}
