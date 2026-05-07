# Apache Answer Codex Instructions

## Project Overview

- Apache Answer is a Q&A platform.
- Backend: Go 1.24 with Gin, xorm, and Pacman-style service wiring.
- Frontend: React 18 + TypeScript under `ui/`.
- Package manager: `pnpm`.
- Go module: `github.com/apache/answer`.

## Repo Map

- Backend implementation lives mostly under `internal/`.
- Frontend implementation lives under `ui/src/`.
- `Makefile` is the top-level build entrypoint, but direct commands below are faster for focused validation.
- The worktree may already contain unrelated edits. Do not revert files you did not touch.

## Verified Validation Commands

- `go build ./internal/...`
- `cd ui && npx tsc --noEmit`
- `go test ./internal/service/siteinfo_common -v`
- `go test ./internal/repo/repo_test -run Test_siteInfoRepo_SaveByType -v`
- `go test ./internal/service/rank/...` currently finds no test files. Do not treat it as real regression coverage.

## Rank / Reputation / Permission Model

- User-facing "reputation" maps to backend `rank`.
- Threshold checks and role-power bypass logic are centered in `internal/service/rank/rank_service.go`.
- Rank mutation, daily limit handling, and `daily_rank_limit` reads are in `internal/repo/rank/user_rank_repo.go`.
- Rank threshold values are config keys read through `ConfigService.GetIntValue(...)`.
- Default rank-related config seeds live in `internal/migrations/init_data.go`; later changes may also appear in `internal/migrations/v*.go`.
- `/answer/api/v1/permission` is a GET endpoint implemented in `internal/controller/permission_controller.go`.
- Its request shape is defined in `internal/schema/permission.go`. It accepts either `action` as a comma-separated string or `actions[]`.
- The response is a per-action mapping of `{ has_permission, no_permission_tip }`.
- Frontend permission fetching goes through `ui/src/services/client/user.ts`, especially `useUserPermission`.
- UI answer gating also uses `loggedUserRank` in `ui/src/pages/Questions/Detail/components/WriteAnswer/index.tsx`.
- Do not assume a fixed count of restricted operations. Permission and rank keys span `internal/base/constant/privilege.go`, `internal/service/permission/*`, and migration/config keys.
- `internal/service/permission/*` assembles member and object actions. `internal/service/rank/rank_service.go` decides whether the user meets rank thresholds.

## Workflow For Rank / Permission Changes

- When adding or changing a restricted action, trace the full path: permission name/constants, rank threshold config key or migration seed, controller/API exposure, and frontend permission key usage.
- Treat role powers and rank thresholds as separate mechanisms. Role powers can bypass rank checks.
- If a change adjusts thresholds, verify both the default config seed and the user-facing gating path.
- Prefer reading real repo files before assuming older Apache Answer behavior.
