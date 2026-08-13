# Launch ICP problem-interview protocol

Status: protocol and evaluator ready; recruitment and interviews not yet
performed.

This protocol implements `LL-101`–`LL-106` without treating feature enthusiasm,
founder intuition, or a convenient sample as buyer evidence.

## Precommitment

Before recruiting, freeze this protocol version as
`launch-icp-interviews-v1`. Recruit across the four launch-hypothesis roles:

- AI engineer;
- ML/platform engineer;
- AI product lead;
- technical founder who personally evaluates or ships AI systems.

Do not recruit only friends already excited by Learnloom. A qualified interview
requires role fit and a concrete attempt within the recent past to understand,
track, evaluate, or apply a changing technical subject. Record people who fail
those screens, but they do not enter the qualified denominator.

## Consent and data handling

Explain the research purpose, whether notes or audio are captured, who can
access them, the retention period, and how to withdraw. Keep identity, contact
details, recordings, transcripts, and verbatim notes in the approved restricted
research system.

The repository-safe corpus contains only:

- a random non-identifying `participantRef`;
- bounded role, workflow, pain, outcome, frequency, and score codes;
- numeric weekly workaround time;
- payment/workaround and design-partner-interest booleans;
- a non-secret opaque evidence reference.

Never put names, handles, employers, emails, URLs, quotes, topics, or transcript
text into the corpus or release evidence. The evaluator rejects `@` and URLs in
references but that check is not a substitute for human privacy review.

## Interview guide

Ask in this order. Do not demonstrate Learnloom or lead with its proposed
features.

1. “Tell me about the last technical subject you needed to understand or keep
   up with for your work.”
2. “What triggered the need, and what work decision or outcome depended on
   it?”
3. “Walk me through exactly what you did first, then next.”
4. “Where did you look? How did you decide which sources or people to trust?”
5. “What did you save, read, ask, or build? What was abandoned?”
6. “How much time did that take that week? Did you pay for any tool, course,
   newsletter, community, or expert help?”
7. “What was still unclear when you stopped?”
8. “When did a similar need happen before or after that?”
9. “How did you know you understood enough to make the decision or do the
   work?”
10. “What did you have to relearn or reconstruct later?”

Only after the behavioral interview, describe the structured design-partner
program—not a product pitch—and ask whether they would participate in observed
setup plus four weekly outcome conversations. `designPartnerInterest=true`
requires explicit agreement to that program, not “sounds interesting.”

## Stable coding rubric

- `painFrequency`: `daily`, `weekly`, `monthly`, or `less_often`. Weekly gate
  evidence is only `daily` or `weekly` behavior supported by a concrete recent
  example.
- `economicConsequence`: 0 none; 1 inconvenience; 2 delayed/degraded work
  decision; 3 material delivery, revenue, risk, or professional consequence.
- `workaroundMinutesWeek`: the reconstructed time for the concrete week, not a
  general estimate. A substantial workaround is at least 120 minutes weekly or
  actual payment.
- `workflowCode`: `search_and_tabs`, `chat_assistant`,
  `newsletter_or_feed`, `read_later_or_notes`, `course_or_training`,
  `colleague_or_expert`, `mixed`, or `none`.
- `primaryPainCode`: `source_discovery`, `source_judgment`,
  `context_rebuilding`, `learning_sequence`, `retention_and_recall`,
  `application_to_work`, `time_fragmentation`, or `other`.
- `desiredOutcomeCode`: `make_decision`, `ship_system`, `evaluate_risk`,
  `explain_to_others`, `stay_current`, `build_capability`, or `other`.
- `urgencyCode`: `active_now`, `next_quarter`, `when_triggered`, or
  `no_deadline`, derived from the real deadline rather than enthusiasm.
- `reachabilityCode`: `direct_network`, `professional_community`,
  `newsletter_or_creator`, `search_or_content`, `partner_channel`, or
  `unknown`, based on where this person already seeks professional help.
- `purchasingAuthority`: `self_serve`, `expense_budget`, `manager_approval`,
  `procurement`, `no_budget`, or `unknown`, based on the actual buying path.

Do not change these definitions after seeing the first interviews. Version the
protocol if a genuine discovery requires a new code, and do not combine
incompatible versions into one gate report.

## Freeze and evaluate

Copy `docs/research/interview-corpus-v1.template.json` into the restricted
research workspace. Complete at least 15 interviews, set `frozenAt` only after
the batch is locked, then run:

```sh
go run ./cmd/interview-eval \
  -corpus /restricted/launch-icp-interviews-v1.json \
  -require-gate \
  > /approved/evidence/location/launch-icp-report.json
```

The exit gate is exactly the roadmap threshold: at least 15 qualified
interviews, at least 10 with weekly pain, at least five already spending money
or two substantial hours per week, and at least 10 consenting to the structured
design-partner program.

## Persona decision

Only after the frozen aggregate passes, complete
`docs/research/persona-decision-v1.template.md` using the restricted qualitative
evidence. Select exactly one launch persona and one primary job-to-be-done.
Document why each other persona was rejected now, what evidence could reopen
it, and which marketing/product claims must not be generalized to it.

Recruit the first 10 design partners from the selected persona. Participation
consent and outcome-story permission remain separate. Classify their product
accounts as `real_user` with reason `external_design_partner` before their
behavior enters launch evidence.

Passing evaluator fixtures proves only that the process is computable. It does
not prove interviews, consent, persona selection, or recruitment happened.
