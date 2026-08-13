# Learnloom paid design-partner beta runbook

Status: ready to operate; recruitment and observation evidence not yet collected.

This runbook implements Phase 10 without weakening its four-week observation
window. It is an operating protocol, not evidence that interviews, payments, or
retention have happened.

## Cohort and consent

Recruit 20–30 people who match the launch persona: AI/software professionals
who need to stay current on a defined technical area and turn updates into
usable professional judgment. Do not add general “curious learners” to make the
cohort larger.

Before observation, explain:

- what setup behavior and learning milestones Learnloom records;
- that the founder will observe setup and request one weekly outcome interview;
- that participation and outcome-story permission are separate choices;
- where qualitative notes are kept and who can access them;
- how to withdraw research consent without losing ordinary product access.

Record consent time in `design_partner_participants`. Store interview notes in
the approved restricted research system; the product database stores only an
opaque `notes_reference`, stable codes, and bounded scores. Never copy a
transcript, email address, lesson text, topic, or source URL into analytics.

Before inserting the participant record, use the restricted identity workflow
to resolve the exact account UUID and run `scripts/classify-evidence-account.sql`
with class `real_user`, reason `external_design_partner`, and a non-secret cohort
evidence reference. The beta report excludes unclassified accounts. Never mark
the founder, a team member, a monitoring probe, or a reimbursed test buyer as a
real user.

## Enrollment state machine

```text
invited -> enrolled -> active -> completed
    |          |          |
 declined   withdrawn   churned
```

- `invited`: selected persona match; no observation yet.
- `enrolled`: explicit research consent recorded.
- `active`: onboarding has begun.
- `withdrawn`: research consent withdrawn; stop future observation/interviews.
- `churned`: stopped using the beta after onboarding.
- `completed`: finished the planned beta window.

Never set `payment_asked_at` before `value_cycle_completed_at`; migration 33
also enforces this invariant.

## First session: observe before coaching

Ask the participant to start with the last technical topic they genuinely tried
to follow. Say only: “Use Learnloom as you naturally would. Please think aloud.”

Do not explain navigation or the source portfolio unless they are blocked. If
you intervene, set `coached=true` and one blocker code:

- `authentication`
- `intent_clarity`
- `source_portfolio`
- `generation_wait`
- `lesson_quality`
- `retrieval_flow`
- `navigation`
- `billing`
- `other`

The setup observation ends after the first lesson is opened, or when a real
blocker prevents progress. Do not mark “success” because the founder finished
the setup for them.

## Weekly cadence (minimum four weeks)

Every week:

1. Run `scripts/design-partner-beta-report.sql` and
   `scripts/billing-economics.sql` in an authorized read-only session.
2. Hold outcome interviews. Ask what decision or work changed, what was not
   worth the time, what they did outside Learnloom, and whether they would be
   disappointed to lose the product. Do not lead with feature requests.
3. Sample source portfolios and lessons across active participants. Score source
   quality, lesson quality, unsupported claims, citation issues, and time fit in
   `design_partner_quality_samples`.
4. Choose the largest activation or retention constraint supported by behavior,
   interviews, and quality samples. Fix that constraint only.
5. Review failures, latency, generation cost, Paddle fees, support volume, and
   all other known COGS. Do not use model-only margin as the commercial gate.

Record non-model costs in `operational_cogs_events` using an immutable provider
invoice/delivery/support reference. Attribute a cost to an account only when
the provider fact supports that allocation; otherwise leave `account_id` null
and allocate shared costs under a reviewed accounting policy before evaluating
the margin gate. Do not insert descriptions, customer identifiers, or provider
payloads into the ledger.

## Value cycle and payment ask

A completed value cycle requires product evidence of a generated and completed
lesson plus a meaningful retrieval/review action. Confirm the participant can
name a useful professional outcome before setting `value_cycle_completed_at`.

Only then ask for the current design-partner price. Use voluntary payment as
evidence; discounts, founder-paid subscriptions, or reimbursement do not count
toward the gate. Record refusal through the in-product stable taxonomy and keep
verbatim context in the restricted notes system.

## Outcome stories

Outcome-story permission is opt-in and separate from research consent. Before
publishing, show the participant the exact wording, metrics, Dossier, and name or
anonymity choice. Store only the permission timestamp in the product database.
Withdrawal cannot guarantee removal of copies already shared by third parties,
but Learnloom should remove its own future use promptly.

## Weekly decision rule

Continue beta unless all commercial gates pass. Do not compress the week-four
window, substitute signup volume for repeated learning, or scale paid traffic.

The Phase 10 exit gate requires:

- at least 30 paying customers;
- week-four repeated learning value;
- paid churn and contribution margin within the roadmap gates;
- representative COGS below 25% of recognized subscription revenue;
- source/lesson quality and reliability gates continuing to pass.

If a gate fails, record the dominant constraint and remain in beta.

## Operator data entry

Use a least-privilege database role and a reviewed statement. Resolve the exact
account in a separate authorized support workflow; do not paste identity data
into tickets or shell history.

Example enrollment after consent (parameters shown symbolically):

```sql
INSERT INTO design_partner_participants (
  account_id, cohort_label, status, persona_match,
  research_consent_at, created_at, updated_at
) VALUES (
  :account_id, :cohort_label, 'enrolled', true, now(), now(), now()
)
ON CONFLICT (account_id) DO UPDATE SET
  cohort_label = EXCLUDED.cohort_label,
  status = 'enrolled',
  persona_match = true,
  research_consent_at = EXCLUDED.research_consent_at,
  consent_withdrawn_at = NULL,
  updated_at = now();
```

The beta tables deliberately have no public or authenticated user API. Cohort
membership, research consent, and editorial quality verdicts are operator-owned
research state, not user-editable product preferences.
