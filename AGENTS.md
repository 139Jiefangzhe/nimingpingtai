# Repository Guidelines

## Project Structure & Module Organization
Apache Answer is split into a Go backend and a React/TypeScript frontend. Backend entrypoints live in `cmd/`, core services and repositories are under `internal/`, reusable packages are in `pkg/`, and plugin integration code is in `plugin/`. Frontend code lives in `ui/src/`, with plugin UI packages under `ui/src/plugins/`. Deployment assets are in `deploy/`, `charts/`, and `docker-compose.yaml`. Generated API docs and rollout notes live in `docs/`.

## Build, Test, and Development Commands
- `make build`: generate code and build the `answer` backend binary from `cmd/answer`.
- `make ui`: install frontend dependencies and build the production UI bundle.
- `make lint`: run ASF header checks and `golangci-lint`.
- `go build ./internal/...`: fast backend compile check for touched packages.
- `go test ./internal/repo/repo_test`: run the repo-backed Go test suite used by the project.
- `cd ui && pnpm lint`: run ESLint autofixes for frontend files.
- `cd ui && npx tsc --noEmit`: run a strict TypeScript check without building assets.

## Coding Style & Naming Conventions
Follow `.editorconfig`: UTF-8, LF endings, and spaces for indentation. Format Go code with `gofmt`; keep package names lowercase and exported identifiers in PascalCase. React components use PascalCase filenames, hooks start with `use`, and shared utilities stay in camelCase modules. Respect existing API path prefixes like `/answer/api/v1/...` when adding routes.

## Testing Guidelines
Place Go tests in `*_test.go` files beside the code they exercise. Prefer focused service or repo tests over broad end-to-end changes. For frontend work, pair `pnpm lint` with `npx tsc --noEmit`; add Testing Library coverage when introducing new interactive behavior. Validate the smallest affected surface first, then run broader checks before opening a PR.

## Commit & Pull Request Guidelines
Recent history follows Conventional Commit style such as `fix: preserve raw vote object ids`, `feat: merge community nav`, and `docs: record rollout`. Keep commits scoped and imperative. PRs should include a short problem statement, the chosen approach, validation commands run, linked issues, and screenshots for UI changes.

## Security & Configuration Tips
Do not commit real secrets. Use `.env.example` as the template for local configuration, and keep WeCom, database, and vault credentials in environment variables or deployment tooling only. Review changes to auth, callback, and connector flows carefully because they affect enterprise login paths.
