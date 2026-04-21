# Curfew Agent Guide

`curfew` is a local-first Go CLI/TUI that enforces terminal quiet hours through managed shell hooks. Keep changes simple, explicit, and easy to validate.

## Product Constraints

- Stay local-first. Do not add telemetry, accounts, background services, or network dependencies.
- Keep `curfew check` fast and predictable. It runs on the interactive shell hot path.
- Treat shell hooks as thin adapters. Core policy decisions belong in the Go binary.
- Preserve clear, boring rule semantics over clever parsing.
- Be careful with runtime and history accounting. Streaks, nightly rollups, and adherence events are product behavior.

## Codebase Map

- `cmd/` owns the Cobra CLI surface.
- `internal/app/` owns product behavior and command orchestration.
- `internal/config/` owns config parsing and validation.
- `internal/friction/` owns override flows and challenge behavior.
- `internal/rules/` owns command matching semantics.
- `internal/runtime/` owns lightweight on-disk session state.
- `internal/schedule/` owns quiet-hours evaluation.
- `internal/shell/` owns shell snippets and managed rc block behavior.
- `internal/store/` owns SQLite-backed history and stats persistence.
- `internal/tui/` owns the Bubble Tea interface.
- `assets/` holds README and release assets.

## Working Style

- Prefer the smallest change that fixes the root cause.
- Preserve existing CLI/config behavior unless the task explicitly changes it.
- Update docs when commands, config shape, install flow, or TUI behavior change.
- If you touch shell integration, use a temporary `HOME` or XDG directory when possible so you do not mutate real dotfiles during validation.
- Do not revert user changes you did not make.

## Validation

- Run focused tests while iterating when helpful, then finish with:
  - `go test ./...`
  - `go build ./...`
- If shell integration changes, also validate the relevant install/init/doctor flow in a safe temporary environment.
- Do not mark work complete without proving the result works or clearly stating what could not be verified.

## Git Conventions

- Branch naming: `raj--<feature_area>--<something>`
- Commit format: `:emoji: verb[feature_area]: brief description`

### Emoji Map

- `:sparkles:` for features
- `:bug:` for fixes
- `:books:` for docs
- `:recycle:` for refactors
- `:wrench:` for chores
- `:mag:` for small cleanup
- `:test_tube:` for tests
- `:zap:` for performance
- `:art:` for formatting/style

## Commit Hygiene

- When the agent creates or amends a commit in this repo, always include the trailer:

```text
Co-authored-by: OpenAI Codex <codex@openai.com>
```

- Keep that trailer on follow-up amend commits too, unless the user explicitly asks not to include it.
- Do not use `git reset --soft` plus `git add -A` to squash work. That can accidentally pull in unrelated changes.

## Pull Requests

- PR titles should use the same emoji-and-verb format as commits.
- Keep PR descriptions brief and focused on user-facing behavior.
- Mention the type of testing performed.
- Include screenshots or terminal captures when TUI or shell UX changes materially.
