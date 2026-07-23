# Learnloom UX revamp brief

Prepared for the upcoming design pass on 23 July 2026.

## Executive summary

Learnloom's strongest product idea is not "manage recurring AI newsletters." It is:

> Turn a subject someone cares about into a continuous learning practice that helps them understand, remember, and build on what they learn.

The current UI is visually coherent, but its interaction model still centers the system:

- streams, schedules, source health, generation stages, delivery states, and publishing controls;
- settings and operational status before the learner's next meaningful action;
- a static archive after a lesson is generated.

The redesigned UX should center the learner:

- **What is worth my attention now?**
- **How does this connect to what I already learned?**
- **Did I understand and retain it?**
- **What progress am I making?**
- **Can I trust the evidence?**
- **Where can I find and use this knowledge later?**

The primary shift is from a **stream management dashboard** to a **guided learning home**.

## Product north star

The ideal recurring loop is:

1. The learner chooses an intent.
2. Learnloom establishes a credible information environment.
3. Learnloom selects a worthwhile lesson.
4. The learner understands it.
5. The learner actively retrieves or applies it.
6. Learnloom records what was covered and what needs reinforcement.
7. The next lesson visibly builds on that history.

Every primary screen should make one of these steps easier. Operational details should be available, but should not dominate the experience.

## Who the redesign should serve

### Primary user

A curious professional or independent learner who:

- follows two to five changing subjects;
- values source quality and nuance;
- has limited time and does not want another feed;
- wants continuity without manually building a curriculum;
- returns several times per week, often from email or mobile;
- cares more about becoming capable than about collecting content.

### Important modes of use

- **Set up:** "I want Learnloom to understand what I am trying to learn."
- **Daily return:** "Tell me what deserves my attention now."
- **Read:** "Help me form a useful mental model without losing the evidence."
- **Recall:** "Help me discover whether I actually understood it."
- **Revisit:** "Find the idea, example, or source I encountered earlier."
- **Tune:** "Adjust the curriculum when it becomes too basic, too broad, noisy, or demanding."
- **Share:** "Publish selected work without accidentally exposing everything."

## Current journey audit

### 1. Marketing to sign-up

#### What works

- The emotional promise is distinctive: a learning home, continuity, durable understanding, and a personal address.
- The product is positioned against feeds and disposable summaries.
- Trust, source visibility, recall, and the archive are presented as core value.

#### Friction and risk

- The marketing preview and the product use different information architecture and vocabulary. Marketing shows "Knowledge Dossiers," global search, "Learning history," and progress bars; the product opens on "Learning streams" and does not provide those global destinations.
- "Claim your learning home" makes the personal site feel like the onboarding outcome, but site setup is a floating, optional control after sign-up.
- The marketing site promises a searchable archive and claim-level source attachment. The current private and public archive experiences do not provide global search, and lesson prose does not expose claim-level citations.
- The marketing preview can set expectations for features or organization that the signed-in product does not yet have.

#### Design direction

- Make the signed-in product feel like the product that was marketed.
- Use one stable vocabulary across marketing, onboarding, email, app, and public site.
- If a capability is not shipping with the redesign, do not make it a prominent preview promise.

### 2. First-run and stream creation

#### What works

- The three-step structure is understandable.
- "What should become clearer?" is a strong intent-first question.
- Learner level, desired progress, source policy, rhythm, and delivery are sensible inputs.
- The review summary reduces uncertainty before creation.

#### Friction and risk

- A new learner must supply a valid source URL before experiencing any value when discovery is unavailable. This is a high-friction expert task at the worst point in the funnel.
- The optional "What would progress feel like?" field is actually central to personalization, but its wording may be too abstract for many users.
- Source entry has no visible validation, preview, detected type, coverage feedback, or indication of what Learnloom will monitor.
- The flow says settings can be changed later, but the current stream page does not expose editing for topic, goal, learner level, lesson length, schedule time, time zone, or source membership in a coherent way.
- "Daily" is effectively the only cadence even though the product principle says quality should win over forced frequency.
- Advanced settings contain major trust decisions—AI Exploration and public visibility—that deserve clear consequences, not simply lower visual priority.
- Creating the stream is not yet the first value moment. The user lands on a system-status page and must choose whether to manually create the first lesson.

#### Design direction

Use progressive commitment:

1. **Learning intent:** topic/question, current familiarity, and a concrete outcome.
2. **Information environment:** choose guided discovery, trusted sources, or hybrid; validate sources immediately and suggest gaps.
3. **Rhythm:** frequency, available time, delivery, and a plain-language privacy summary.
4. **Confirmation:** show what Learnloom understood and offer a clear primary action: **Prepare my first lesson**.

When automated discovery is unavailable, design an honest fallback:

- one URL is enough;
- explain accepted URL types;
- validate it inline;
- show what was detected;
- allow "I need help choosing sources" as a request or waitlist path instead of a dead end.

### 3. Home and daily return

#### What works

- The current dashboard is calm and scannable.
- Active and paused streams are easy to distinguish.
- Search and filters will help once a user has many streams.
- The "Next in your rhythm" module starts to prioritize a return action.

#### Friction and risk

- The home page primarily answers "What streams do I own?" rather than "What should I do now?"
- "Next in your rhythm" selects the first active stream, not necessarily the soonest, newest, unread, or most useful lesson.
- Counts for streams, active streams, and total lessons are system inventory. They do not show learning momentum, unread work, concepts covered, or recall due.
- Stream cards repeat configuration and archive counts, while ready lessons and in-progress lessons are not first-class objects.
- There is no global lesson archive, global search, saved material, or learning-history view despite those being central to the promise.
- Mobile becomes a long stack of large cards with repeated metadata and calls to action.
- There is no clear account, notification, email, or general settings destination.

#### Design direction

Make the default home a **Today** view:

- **Continue learning:** resume the in-progress lesson at the previous position.
- **Ready for you:** the strongest new lesson, with why it was selected and expected time.
- **Recall due:** one to three concepts worth retrieving.
- **Coming next:** transparent schedule or "waiting for stronger evidence."
- **Recent progress:** a meaningful weekly narrative, not a vanity count.

Keep stream management available as a separate destination.

### 4. Stream overview and tuning

#### What works

- The page exposes source health, the learning blueprint, schedule, generation state, and lesson history.
- Manual creation, pause/resume, email control, AI Exploration, and publication control are discoverable.

#### Friction and risk

- The page gives more space to the generation pipeline and source operations than to the latest lesson or continuity.
- Six pipeline stages describe the system's internal process but do not help most learners decide what to do.
- The learning archive is pushed below several large configuration cards.
- The latest lesson does not receive a clear hero treatment with unread/read/in-progress state.
- Sources are inspectable but not editable. Learners cannot repair, remove, add, prioritize, or explain why a source is trusted.
- The blueprint is descriptive but not tunable.
- Settings are fragmented across header actions, a settings icon, inline toggles, per-lesson actions, and a floating personal-site control.
- Public visibility has three layers—site, stream, and Dossier—but the hierarchy and resulting audience are not explained.
- Status language such as "generation," "trigger," and "Issue" exposes implementation concepts.
- Table titles are truncated and the row is optimized for operational status rather than later retrieval.
- The page does not explain why a lesson was selected, how it connects to prior lessons, or what the system learned about the learner.

#### Design direction

Structure a stream around four clear areas:

- **Overview:** next/ready lesson, current direction, continuity, and a small health summary.
- **Lessons:** readable chronological curriculum with search/filter and progress state.
- **Sources:** add, remove, validate, prioritize, and understand coverage.
- **Settings:** intent, level, outcome, cadence, delivery, AI policy, privacy, pause, and destructive actions.

Collapse pipeline detail into a secondary "How this lesson was prepared" status. Show it only while work is underway or when intervention is required.

### 5. Waiting for generation

#### Friction and risk

- After queuing a lesson, the client reloads once but does not visibly establish ongoing progress or automatic completion.
- The learner sees a generic queued notice without an estimate, permission to leave, or a reliable completion path.
- A static pipeline can imply precision without reflecting real progress.

#### Design direction

Design an honest asynchronous state:

- "Preparing your lesson; you can leave this page."
- expected range rather than a fake exact percentage;
- what is happening in learner language;
- automatic refresh or push completion;
- email/browser notification option;
- recovery state if sources fail;
- a useful alternate action while waiting, such as a short starter concept or reviewing the prior lesson.

### 6. Lesson reading and active learning

#### What works

- The reader has strong hierarchy, a clear objective, sections, a lesson map, retrieval prompts, application, and visible sources.
- Read time and learner level set useful expectations.
- The experience feels more like a prepared lesson than an AI chat response.

#### Friction and risk

- The reader is a static article. It does not track progress, save position, mark completion, or invite reflection.
- Retrieval questions are displayed immediately and do not support answering, revealing, self-rating, or later review.
- There is no spaced-retrieval queue even though remembering is part of the product promise.
- Sources are collected in a sidebar, but claims in the body are not visibly traceable to specific evidence.
- There is no "why this lesson now," explicit connection to prior learning, prerequisite link, or next conceptual step.
- There is no next/previous lesson navigation.
- The breadcrumb is labeled "Lesson 01" rather than the actual position or date.
- The normal application chrome and floating personal-site control compete with the focused reading experience; in the current capture the floating control overlaps lesson navigation.
- There are no learner tools such as save, note, highlight, copy citation, report a weak claim, or adjust difficulty.

#### Design direction

Treat the reader as a learning session with three phases:

1. **Orient:** why this now, objective, prior connection, time commitment.
2. **Understand:** focused content, evidence attached to claims, lightweight navigation, optional source details.
3. **Consolidate:** answer before reveal, confidence rating, application/reflection, completion, and what comes next.

Required behaviors:

- autosave position and responses;
- resume state across devices;
- mark complete without forcing every interaction;
- hide/reveal retrieval answers and record confidence;
- schedule selected concepts for later retrieval;
- show inline citation markers with source detail on demand;
- allow "too basic," "too advanced," "not relevant," and "question this claim" feedback;
- offer previous/next and return-to-stream actions at the end.

### 7. Library, search, and long-term progress

#### Friction and risk

- The durable archive is described as the product's final destination, but the private app only exposes lesson history inside each stream.
- Search on the dashboard searches stream names and topics, not lesson content, concepts, sources, or retrieval questions.
- There is no cross-stream timeline, concept index, source index, saved collection, or review history.
- Total lesson count cannot tell a learner what they now understand.

#### Design direction

Create a first-class **Library**:

- search across titles, full lesson content, concepts, sources, and notes;
- filter by stream, date, read state, saved state, source, and confidence;
- group by chronological archive or topic;
- show where a concept first appeared and where it was revisited;
- resume unread/in-progress lessons;
- expose saved highlights and learner notes if those capabilities are included.

Create a lightweight **Progress** model:

- themes explored;
- concepts introduced and revisited;
- lessons completed or intentionally skipped;
- retrieval confidence over time;
- a weekly learning recap written in plain language.

Avoid punitive streaks. The emotional goal is steady capability, not guilt.

### 8. Personal site and publishing

#### What works

- A personal subdomain is distinctive and gives the archive lasting identity.
- Site, stream, and per-Dossier controls support granular privacy.

#### Friction and risk

- The personal site is positioned as the main outcome on marketing, then reduced to a floating bottom-corner widget in the app.
- Claiming the site is disconnected from the moment when a learner has something worth publishing.
- The three visibility layers are individually controllable but not understandable as one publishing system.
- The current site setup only asks for address and display name; it does not let the learner preview or shape the public identity described on marketing.
- The public archive is minimal and lacks the marketed navigation and search behavior.
- Per-Dossier "Publish/Hide" actions in an operational table make accidental visibility changes easier to misunderstand.

#### Design direction

Make **Publishing** a coherent destination, not a floating utility:

- preview the exact public result before publishing;
- show a visibility ladder: site → stream → lesson;
- state the effective audience for every item;
- use a review step before the first public publish;
- let the learner choose address, display name, description, and basic profile identity;
- surface setup contextually after the first completed lesson;
- provide a share action on a published lesson with a clear private/public state.

## Recommended information architecture

### Primary navigation

1. **Today**
   - Continue learning
   - Ready lessons
   - Recall due
   - Coming next
2. **Streams**
   - Active and paused streams
   - Stream overview, lessons, sources, settings
3. **Library**
   - Global archive and search
   - Saved items and notes, if supported
4. **Review**
   - Retrieval queue
   - Confidence and concepts to revisit
5. **Publishing**
   - Personal site setup and preview
   - Site, stream, and lesson visibility
6. **Account settings**
   - Profile, email, notifications, default rhythm, privacy, and account actions

On mobile, prioritize Today, Streams, Library, and Review in the primary navigation. Publishing and settings can live under the account menu.

## Proposed end-to-end journey

### Activation

1. Sign up.
2. See a one-screen value orientation with a sample Dossier or annotated preview.
3. Describe a learning intent.
4. Confirm level and concrete outcome.
5. Choose or establish sources with immediate validation.
6. Choose a sustainable rhythm and delivery.
7. Review Learnloom's interpretation.
8. Prepare the first lesson.
9. While waiting, show what will happen and let the learner leave safely.
10. Notify when ready.

**Activation event:** the learner completes the first lesson's consolidation step, not merely creates a stream.

### Recurring use

1. Open from email, notification, or Today.
2. Understand why the lesson is worth attention.
3. Read or resume.
4. Complete at least one retrieval or application action.
5. See the connection recorded in Learning History.
6. Know what is likely to happen next.

### Re-engagement

When a learner has missed several lessons:

- do not present a guilt-inducing backlog;
- summarize what changed;
- recommend one best re-entry lesson;
- offer to reduce frequency, pause, or reset the direction;
- let the learner archive unneeded unread lessons in one action.

## P0 design requirements

These should shape the first redesign concepts.

1. **Today-first home** centered on the next learning action.
2. **First-value onboarding** that ends in preparing and completing a first lesson.
3. **Editable stream setup** for intent, level, outcome, cadence, sources, delivery, and privacy.
4. **Stream page hierarchy** that prioritizes lessons and continuity over pipeline diagnostics.
5. **Active lesson session** with resume, completion, retrieval, prior connection, and next step.
6. **Claim-level evidence interaction** or a clearly scoped alternative that does not overpromise.
7. **Global Library and search** across generated learning, not only stream names.
8. **Unified publishing model** with effective visibility and preview.
9. **Honest asynchronous states** for preparing, delayed, insufficient-evidence, and failed lessons.
10. **Responsive density** designed for daily mobile use, not desktop cards stacked vertically.

## P1 opportunities

- Weekly learning recap.
- Learner feedback that tunes difficulty and relevance.
- Cross-stream concept connections.
- Saved highlights and notes.
- Recall queue with confidence.
- Source coverage recommendations and source repair.
- Gentle re-entry after inactivity.
- Notification preferences by stream.
- "No lesson today" state when evidence is weak.

## P2 opportunities

- Shareable learning paths or curated collections.
- Export of notes, highlights, or completed curriculum.
- Personal-site themes and richer identity.
- Collaborative or mentor modes.
- Rich concept maps after enough history exists.

## Required states for the design agent

Do not design only populated happy paths. Include:

- brand-new account;
- onboarding abandoned and resumed;
- invalid, duplicate, unsupported, and unreachable source;
- discovery unavailable;
- stream created, first lesson not started;
- generation queued, taking longer than expected, and failed;
- insufficient worthwhile evidence today;
- lesson ready, in progress, completed, and skipped;
- no retrieval due and several reviews overdue;
- one stream and many streams;
- paused stream;
- source needs attention;
- no search results;
- personal site unclaimed, private, public, and partially published;
- offline or request error with recovery;
- mobile reader, mobile source setup, and mobile archive.

## Interaction and content principles

### Use learner language

Prefer:

- lesson;
- ready for you;
- prepared from;
- scheduled;
- needs your attention;
- source unavailable;
- continue learning;
- review this concept.

Avoid exposing:

- Issue;
- trigger;
- generation status;
- worker;
- enrichment;
- quality gate;
- publication state.

Those terms can exist in support or advanced diagnostics.

### Explain consequences at the point of choice

- AI Exploration: what is synthetic, how it is labeled, and when it appears.
- Public visibility: exactly who can access the result.
- Pause: whether prepared lessons remain and what happens to the next schedule.
- Source removal: whether existing lessons and citations are retained.
- Difficulty change: whether it affects future lessons only.

### Respect attention

- Do not turn missed lessons into a red backlog.
- Do not force novelty to keep a schedule.
- Keep one dominant action per screen.
- Prefer progressive disclosure for pipeline and delivery diagnostics.
- Preserve a distraction-free reading mode.

### Accessibility baseline

- Body text contrast of at least 4.5:1.
- Non-text interactive contrast of at least 3:1.
- Visible focus styles and full keyboard navigation.
- Minimum 44px touch targets on mobile.
- Do not use color alone for source health, status, or visibility.
- Respect reduced-motion preferences.
- Ensure retrieval, source detail, and publishing controls work with assistive technology.
- Avoid very small uppercase metadata as the only carrier of important information.

## Success metrics

### Activation

- Sign-up → stream setup completion.
- Stream setup → first lesson requested.
- First lesson requested → first lesson opened.
- First lesson opened → consolidation completed.
- Time to first lesson and abandonment by setup step.

### Recurring value

- Weekly active learners who complete or intentionally skip a lesson.
- Resume rate for in-progress lessons.
- Retrieval participation and later return rate.
- Percentage of lessons rated appropriately difficult and relevant.
- Search success: query followed by opening a result without immediate reformulation.
- Re-entry rate after seven or more inactive days.

### Trust and control

- Source validation failure rate.
- Rate of source repair after an alert.
- "Question this claim" rate and resolution path.
- Accidental publish reversals within a short window.
- Understanding of effective visibility in usability testing.

### Guardrails

- Unsubscribe or pause rate after increasing cadence.
- Unread backlog growth.
- Lesson generation without open/read behavior.
- Public sharing of content the learner believed was private.

## Questions to validate with users

1. When users say they want to "keep up" with a topic, do they want daily lessons, event-driven lessons, or a weekly synthesis?
2. Will target users bring trusted URLs, or do they expect Learnloom to establish the source set?
3. What is the smallest first lesson that proves value without feeling like a generic AI response?
4. Do learners want to answer retrieval questions inside Learnloom, or simply use them as prompts?
5. What would make continuity believable: explicit references to prior lessons, a visible path, progress summaries, or all three?
6. Is the personal site a primary motivation, a later reward, or relevant only to a subset of users?
7. Which private-learning tools matter most: search, highlights, notes, saved items, exports, or concept review?
8. How much system transparency builds trust before it becomes operational noise?

## Design-agent assignment

Use this as the working prompt:

> Redesign Learnloom as a guided learning home, not a stream-management dashboard. Start with the complete journey from sign-up to first completed lesson, then the recurring Today → Lesson → Recall → History loop. Create desktop and mobile concepts for Today, onboarding, stream overview, active lesson, global Library, Review, and Publishing. Preserve Learnloom's calm, thoughtful character, but make the hierarchy behavioral: the next meaningful learning action should dominate each screen. Move generation, delivery, and source diagnostics behind progressive disclosure unless intervention is required. Show all P0 states, especially waiting, failure, insufficient evidence, resume, completion, and privacy. Use one consistent learner-facing vocabulary. Treat source trust, continuity, retrieval, and effective visibility as interaction problems, not decorative labels.

## Evidence reviewed

- Product positioning and principles: `launch-video/PRODUCT-BRIEF.md`
- Marketing journey: `web/src/MarketingLanding.jsx`
- Authentication and personal-site setup: `web/src/HostedApp.jsx`
- Dashboard and navigation: `web/src/App.jsx`
- Stream creation: `web/src/NewsletterCreate.jsx`
- Stream overview and archive: `web/src/NewsletterDetail.jsx`
- Lesson reader: `web/src/IssueDetail.jsx`
- Public archive: `internal/httpapp/reading.go`
- Current UI captures: `screenshots/current-ui-2026-07-23/`

