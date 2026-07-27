# Repository Guidelines

## Project Structure & Module Organization

Learnloom is a hosted learning service with a Go backend and React/TypeScript frontend. `cmd/learnloom/` contains the binary entry point and its `web`, `worker`, and `migrate` roles. Backend packages live under `internal/` (for example, `store`, `source`, `execution`, and `dossier`); SQL migrations are in `internal/store/migrations/`. The Vite application is in `web/src/`, with static assets in `web/public/`. Operational and architecture guidance lives in `docs/`, while deployment infrastructure is defined by `Dockerfile`, `compose*.yaml`, and `infra/`. Launch-video tooling is isolated under `launch-video/`.

## Build, Test, and Development Commands

- `npm ci` installs the locked Node 24+ dependencies.
- `npm run dev` starts the Vite frontend locally.
- `npm run demo` starts Vite with demo mode enabled.
- `npm run check` runs ESLint, TypeScript checks, and the production frontend build.
- `npm test` runs the Vitest suite once.
- `go test ./cmd/... ./internal/...` runs backend tests; add `-race` before release.
- `go vet ./...` performs Go static analysis.
- `docker compose up --build` builds and starts the local service stack.

Copy `.env.example` to `.env` for local development and replace every placeholder.

## Coding Style & Naming Conventions

Format Go code with `gofmt`; use short, lowercase package names and keep implementation inside `internal/`. TypeScript uses two-space indentation, ES modules, React function components, and ESLint rules from `eslint.config.js`. Name components and component files in PascalCase (`TodayPage.tsx`), hooks with a `use` prefix, and utilities in camelCase. Keep modules focused and place tests beside the code they exercise.

## Testing Guidelines

Go tests use the standard `testing` package and `*_test.go`; frontend tests use Vitest and `*.test.ts` or `*.test.tsx`. Add regression tests for behavior changes, especially ownership, source-safety, persistence, and worker retry logic. Database integration tests require `TEST_DATABASE_URL`; see `README.md` for the lifecycle-test command.

## Commit & Pull Request Guidelines

Recent commits use concise, imperative summaries such as `Serve favicon across hosted pages`; keep each commit scoped to one concern. Pull requests should explain the problem and solution, identify configuration or migration changes, link relevant issues, and list verification commands. Include screenshots for visible UI changes and note any deployment, DNS, provider, or secret-management follow-up.

## Security & Configuration

Never commit secrets or populated environment files. Preserve account-ownership checks, CSRF/origin validation, safe source-fetching restrictions, and idempotent delivery behavior. Review `docs/architecture.md` and relevant ADRs before changing service boundaries.
