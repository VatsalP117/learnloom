# Public source rights-holder response

Use this runbook for a credible source-owner or rights-holder request involving
a public Learnloom Dossier. This is an operational hold-and-review procedure,
not a determination that the requester owns the work or that the underlying
use is lawful or unlawful.

## Intake and verification

1. Create a support/legal case with a non-identifying reference such as
   `RIGHTS-1042`. Keep requester identity, correspondence, and any legal
   documents in the approved support system, never in command history, Git, or
   the database moderation reason.
2. Record the exact public URL, affected claim/source, requested remedy, and a
   safe callback channel in that case.
3. Verify the requester through a domain mailbox, published contact method, or
   other evidence appropriate to the claim. Do not ask them to send passwords,
   session tokens, or unnecessary identity documents.
4. If the request is credible or urgent, hold the exact Dossier first. A hold
   removes it from public reading, site listings, sitemaps, follows, analytics,
   and indexing eligibility without deleting the owner's private learning
   record.

## Apply an exact public hold

Use a dedicated active operator Account that is not the Dossier owner and a
database credential limited to executing the reviewed command. Supply secrets
through the operator environment; do not paste them into a ticket or shell
transcript.

```sh
export DATABASE_URL='postgres://...'
export LEARNLOOM_OPERATOR_ACCOUNT_ID='00000000-0000-0000-0000-000000000000'

go run ./cmd/public-hold \
  -username exact-public-username \
  -public-id exact-public-dossier-uuid \
  -confirm-public-id exact-public-dossier-uuid \
  -case-reference RIGHTS-1042 \
  -reason 'Reviewing a verified rights-holder removal request.'
```

The first run validates input only. Review the exact target, then repeat with
`-apply`. The command requires the public ID twice, verifies schema parity,
refuses an owner acting as operator, updates only a currently published exact
Dossier, and atomically writes a `publication_held` audit action. Retrying an
already-held Dossier is safe and does not create another action.

## Investigation and resolution

1. Confirm the public page returns not found and is absent from its path and
   sitemap. Record a dated probe in the case.
2. Notify the Dossier owner without disclosing requester information beyond
   what the requester authorized.
3. Compare the public synthesis and citations with the reported source. Do not
   distribute stored source bodies outside the authorized investigation.
4. Choose and document one outcome: keep held/unpublish, publish a correction,
   revise through a new generation, restore after rejecting the request, or
   escalate to counsel. Restoration uses the owner's authenticated moderation
   controls after the operator/counsel decision is recorded; the emergency
   command intentionally cannot clear a hold.
5. If the request covers future retrieval of a domain or URL, keep the affected
   public material held and apply the narrowest reviewed retrieval policy. Use
   `exact_url` unless the verified request and decision cover the registrable
   domain. Validate first, then repeat with `-apply`:

   ```sh
   go run ./cmd/source-policy \
     -scope exact_url \
     -value 'https://example.com/reported-source' \
     -confirm-value 'https://example.com/reported-source' \
     -action block \
     -case-reference RIGHTS-1042 \
     -reason 'Verified request requires a future-retrieval hold.'
   ```

   The policy is append-only and enforced before initial retrieval and on every
   redirect. Resolution is another audited action with `-action unblock`; do
   not edit or delete the original event. Keep requester identity and legal
   analysis in the approved case system, not in the command arguments.
6. Record final probes, timestamps, decision owner, and any requester/owner
   notifications in the support/legal case. Never put personal details in the
   public correction or moderation reason.

## Required drill before public launch

In staging, run one synthetic rights-holder case end to end: validated intake,
exact hold, public/path/sitemap removal, idempotent retry, owner notification,
documented decision, and authorized restoration or permanent unpublish. A unit
or database test alone does not satisfy this operational launch gate.
