# ADR-0003: Concrete external adapters

Status: accepted

## Context

The existing provider module supports Command Code, OpenAI-compatible HTTP, and
a production demo. Its interface exposes executable paths and local process
behavior. Resend is the only delivery implementation, Clerk is the identity
source, and user-controlled source URLs require one consistent trust policy.

## Decision

- Model completion uses OpenAI-compatible Chat Completions over HTTPS.
- Command Code, process spawning, provider-kind selection, and production demo
  behavior are deleted.
- Resend remains the email delivery implementation.
- Clerk remains the Account identity source; session tokens and webhook
  signatures are verified with official libraries.
- One deep Source Item acquisition module owns public-address validation,
  pinned connections, redirect validation, content-type checks, byte limits,
  timeouts, feed normalization, article extraction, and fallback.

Interfaces for model completion, time, and external delivery exist only as
internal test seams where a test adapter exercises meaningful variation.

## Consequences

- Provider retries, redaction, and response validation have one locality.
- Feed and article retrieval cannot drift into different security policies.
- Production configuration contains no executable or provider-kind fields.
- New production adapters require a new decision rather than speculative seams.
