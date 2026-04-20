# Contributing

Thanks for helping make `curfew` better.

## Local setup

1. Install Go `1.26` or newer.
2. Clone the repo.
3. Run:

```bash
go test ./...
go build ./...
```

If you are working on shell integration, it is worth testing with a temporary home directory so you do not mutate your real dotfiles by accident.

## Project shape

Keep the boundaries simple:

- `cmd/` owns the Cobra CLI surface
- `internal/app/` owns the product behavior and orchestration
- `internal/config/` owns config parsing and validation
- `internal/friction/` owns override flows
- `internal/shell/` owns emitted hook snippets and managed rc blocks
- `internal/store/` owns SQLite history/stats persistence
- `internal/tui/` owns the Bubble Tea interface

## What to optimize for

- Keep `curfew check` fast and predictable. It runs on the interactive hot path.
- Prefer boring, testable rule semantics over clever parsing.
- Keep everything local-first and explicit. No hidden background behavior.
- Treat shell hooks as thin adapters. The Go binary should own the real logic.
- Be careful with runtime/history accounting. Streaks and nightly rollups are part of the product, not just bookkeeping.

## Testing expectations

When behavior changes, add or update tests close to the layer that changed:

- rule matching and schedule logic should get direct unit coverage
- runtime/history changes should get app or store tests
- interactive TUI flows should get model tests
- CLI or binary wiring changes should get end-to-end coverage when practical

Before opening a PR, run:

```bash
go test ./...
go build ./...
```

## Pull requests

- Keep scope tight.
- Explain the user-facing effect, not just the code delta.
- Call out shell-specific caveats if the change affects `zsh`, `bash`, or `fish`.
- Include screenshots or terminal captures when the TUI or shell experience changes materially.
- Update docs when command behavior, config shape, or install flow changes.
