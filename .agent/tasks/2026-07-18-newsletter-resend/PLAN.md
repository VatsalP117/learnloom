# Implementation Plan

## 1. Task Summary

Turn the dashboard test phase into a useful single-user email product while
preserving the separation between expensive Issue generation and external
delivery. Resend remains an installation-level provider credential and sender;
recipients and enablement belong to each Newsletter.

## 2. Current System Understanding

Learnloom already has SQLite Newsletters and Issues, a worker that invokes the
Daily Run with delivery suppressed, a server-rendered dashboard, and a tested
Resend adapter used by the original CLI flow. The Workspace schema is version
1. Issue generation persists a Dossier JSON path and rendered Markdown path,
which are sufficient to deliver later without another model call.

## 3. Scope

### In Scope

- Workspace schema version 2 with per-Newsletter email settings
- Durable Issue Delivery Receipts and atomic queue operations
- A safe version 1 to version 2 migration preserving existing data
- Email settings on Newsletter creation and detail pages
- One installation-wide enabled Resend configuration for credentials/sender
- Worker delivery from persisted Dossier and Markdown artifacts
- Stable idempotency keys and provider email IDs
- Visible pending, delivering, delivered, and failed states
- Manual retry of failed delivery without Issue regeneration
- Sent counts and delivery status in dashboard projections
- Automated migration, concurrency, worker, HTTP, and regression tests
- VM/container deployment documentation and operational diagnostics

### Out of Scope

- Authentication or public internet exposure
- Multiple users, subscriber management, or public unsubscribe flows
- Notion delivery
- Resend webhook event tracking
- Automatic repeated retry/backoff
- Editing Newsletter generation or schedule fields
- Storing API keys in SQLite or the dashboard

## 4. Technical Approach

Migrate the Workspace in-place to schema version 2. Add `email_enabled` and
`email_recipients_json` columns to Newsletters and an `issue_deliveries` table
with one `email` receipt per Issue. Completing generation and enqueueing the
receipt happen in one SQLite transaction, removing the crash gap between those
states. Claim, success, failure, and manual retry are explicit guarded state
transitions and remain safe across multiple workers.

The worker first drains generation work and then delivery work. Delivery loads
the already-persisted Dossier JSON and Markdown, overlays the Newsletter's
recipients onto the enabled installation-level Resend config, and calls the
existing adapter with a stable `newsletter-email` identity. A failed delivery
stays failed until the user retries it, preventing a bad credential or domain
from being hammered each poll cycle.

The dashboard never accepts or displays a Resend key. It exposes recipient
settings, status badges, provider receipt IDs, and a CSRF-protected retry
action. Existing Newsletters migrate with email disabled.

## 5. Execution Plan

1. Record this plan and establish the migration contract.
2. Implement schema v2, Newsletter email validation, Delivery Receipt
   projections, and atomic transitions.
3. Integrate worker delivery from persisted artifacts using the Resend adapter.
4. Add dashboard create/settings/status/retry controls.
5. Update configuration examples and VM/container operating documentation.
6. Run unit, integration, static, container, and secret checks.
7. Package an independent review, fix blocking findings, and open a draft PR.

## 6. Test Plan

- Fresh databases initialize directly at schema version 2.
- Version 1 databases migrate without losing Newsletters or Issues.
- Enabling email requires valid, normalized recipients.
- Existing Newsletters remain email-disabled after migration.
- Completing an Issue atomically creates one pending Delivery Receipt.
- Two SQLite connections cannot claim the same delivery.
- Delivered and failed outcomes persist safe metadata.
- Manual retry only transitions an eligible failed receipt back to pending.
- Worker sends persisted artifacts to Newsletter recipients.
- Delivery retry does not invoke Daily Run again.
- Missing Resend setup produces a visible failed receipt.
- Dashboard mutations require CSRF and render escaped recipient/status data.
- Existing CLI and dashboard behavior remain green.

## 7. Acceptance Criteria

- A user can configure multiple Newsletters with different recipient lists.
- Generated Issues show delivery state independently of generation state.
- A successful Resend call records the provider email ID and sent timestamp.
- A failed send can be retried from the dashboard without spending generation
  tokens again.
- No API key is persisted in SQLite, HTML, logs, or tracked files.
- Existing version 1 workspace data is upgraded safely.
- The full automated suite and deployment checks pass.

## 8. Risks and Guardrails

- Resend requires a verified sender domain; document this and surface failures.
- Provider idempotency is time-bounded, so the local delivered receipt is the
  durable duplicate guard.
- Artifact reads can fail after generation; record a delivery failure rather
  than changing Issue generation status.
- Do not auto-retry failed receipts in this phase.
- Keep the web listener loopback-only until authentication is implemented.
- Validate recipient lengths and addresses before persistence.

