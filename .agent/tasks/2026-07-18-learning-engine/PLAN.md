# Implementation Plan

## 1. Task Summary

Create a dependency-light Node.js CLI for a daily, source-grounded learning
dossier powered by Command Code's DeepSeek V4 Pro subscription allocation.

## 2. Current System Understanding

The repository is empty apart from Git metadata. Node.js 24 is installed.
Command Code is not yet installed. Its official CLI documents a non-interactive
`cmd --print` mode, `--model`, `--list-models`, and authentication status
commands.

## 3. Scope

### In Scope

- JSON configuration for interests, feeds, run limits, and provider settings
- RSS/Atom retrieval and normalization
- Multi-pass researcher, skeptic, teacher, and examiner prompts
- Command Code headless provider plus deterministic demo provider
- Markdown dossier and local JSON learning history
- CLI commands for init, run, doctor, and macOS scheduling
- Unit and integration tests using only local fixtures
- Setup and operating documentation

### Out of Scope

- Scraping private or authenticated sources
- Credential extraction or undocumented Command Code APIs
- Email, Slack, or Telegram delivery
- A hosted service or graphical interface
- Automatic evaluation of a learner's free-form answers

## 4. Proposed Technical Approach

Use Node.js ESM and built-in APIs only. Keep modules small and independently
testable. Call Command Code by argument array (never a shell) using its
documented print mode, one turn, plan permissions, and a configurable model.
Limit source and intermediate text before invocation. Store generated artifacts
under ignored local directories and write files atomically.

## 5. Step-by-Step Execution Plan

1. Establish repository metadata, task artifacts, package scripts, and docs.
2. Implement configuration validation and feed parsing/fetching.
3. Implement provider adapters and staged prompt pipeline.
4. Implement persistent history and Markdown output.
5. Implement CLI commands and launchd scheduling.
6. Add fixtures and tests for parsing, pipeline, persistence, and scheduling.
7. Install/check Command Code, validate its live model identifier, and run the
   safe doctor command.
8. Run checks, create a review package, review, fix blockers, and document.

## 6. Test Plan

- Node test runner unit tests for RSS and Atom parsing
- Configuration validation tests
- Prompt/pipeline integration test with deterministic demo provider
- Schedule plist generation test
- CLI demo run that performs no network or model calls
- Syntax/check command across source files
- Command Code status/model discovery checks when installed

## 7. Acceptance Criteria

- A new user can run `npm run demo` and receive a complete dossier.
- `npm run doctor` explains provider readiness without exposing credentials.
- A live run uses `deepseek-v4-pro` through `cmd --print`.
- The dossier includes sources, teaching material, critique, and questions.
- Consecutive runs receive prior learning history.
- The scheduler can install a 9:00 a.m. launch agent.
- Tests and checks pass.

## 8. Risks and Guardrails

- Feed XML is diverse: support common RSS/Atom forms and skip malformed items.
- Headless CLI output may vary: treat stdout as text and surface stderr safely.
- Subscription model availability may change: discover it with
  `cmd --list-models` and keep the model configurable.
- Never read or copy Command Code authentication files.
- Never invoke the provider with write permissions.
- Never include secrets in generated artifacts or Git.

## 9. Executor Instructions

Implement thin vertical slices and commit after repository setup, core pipeline,
and operational integration. Keep dependencies at zero unless a concrete
correctness problem makes one necessary. Do not install the daily schedule
until a demo run and provider diagnostics succeed.

