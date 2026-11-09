# AGENTS.md — Trac2GitLab

This repository contains a Go migration engine (`trac2gitlab`) and the agent
skills that orchestrate it. User documentation lives in `docs/`; agent
instructions live here and in `skills/`.

## Hard rules

1. **Source data is read-only.** Never write to Trac. Never push to or alter the
   source Mercurial repository. Clone/pull only.
2. **No secrets in the repo.** Tokens and passwords come from
   `projects/<name>/.env` or the environment, never from `migration.yaml`,
   mappings, state, reports, docs, or code.
3. **Projects are local.** Everything under `projects/` (except `.gitkeep`) is
   gitignored. Never commit project data.
4. **No destructive GitLab operations without explicit confirmation.**
   Overwriting a non-empty repo, existing issues, or wiki pages requires a clear
   user confirmation. Dry-run first.
5. **Historical metadata is mandatory.** Preserve Trac issue/comment
   timestamps, close dates where supported, and historical authors through
   `mappings/users.yaml` + the `patch` step. Never fall back to the migration
   date or migration author silently.
6. **Preflight gates everything.** No target mutation before
   `trac2gitlab preflight` passes. Required access is **Owner or instance
   Administrator**; Maintainer is insufficient. Historical authors also need
   shell/console access for the one-time `patch` step. There is no
   `--force`/`--ignore-permissions` bypass.
7. **Never guess identities.** Unmapped authors stay unresolved and get an
   explicit `Originally by ...` footer. Do not attribute their content to the
   migration account.
8. **Never silently drop content.** Unsupported macros, missing attachments, and
   unconvertible formatting become warnings in reports.
9. **Resumability is mandatory.** Every mutating phase records progress in
   `state/migration.yaml` and mapping files. Rerun only failed/pending work.
10. **Fixtures before production.** Test converters/importers on fixtures and a
    non-production project first.

## Division of labor

The agent orchestrates. The Go engine transforms.

| Agent | Engine |
|---|---|
| ask, approve, choose target | export Trac over XML-RPC |
| enforce preflight and confirmations | convert TracWiki → Markdown |
| interpret warnings and failures | rewrite references via explicit maps |
| resume failed work | create issues/notes/uploads/wiki |
| produce human reports | state, validation, author patch script |

Do not hand-rewrite thousands of records in the LLM. Add or fix the deterministic
engine code instead.

## Agent-driven migration flow

All commands take `--project-dir projects/<name>`.

1. `trac2gitlab init --project-dir ...`
2. User fills `migration.yaml` + `.env`; propose `mappings/users.yaml`.
3. `trac2gitlab preflight --project-dir ...` — must be `READY` or
   `READY WITH WARNINGS` accepted by the user. `BLOCKED`/`NOT READY` stops here.
4. `trac2gitlab users validate --project-dir ... --check-gitlab` — resolve
   identities; unresolved users need a user decision.
5. `trac2gitlab export --project-dir ...`
6. `trac2gitlab migrate --project-dir ...` — dry-run. Show the plan.
7. Get explicit confirmation, then
   `trac2gitlab migrate --project-dir ... --execute`.
8. `trac2gitlab patch --project-dir ...` — emit the dry-run block. The user runs
   it on the GitLab host. Review, then emit `--apply` and run it.
9. `trac2gitlab validate --project-dir ...` — report `PASS`,
   `PASS WITH WARNINGS`, or `FAIL`.
10. Summarize: counts, warnings, unresolved identities, remaining manual work.

The detailed skill contracts live in `skills/`.

## File contract

```text
projects/<name>/
├── migration.yaml        # human-edited config (no secrets)
├── .env                  # credentials, gitignored
├── mappings/
│   ├── users.yaml
│   ├── trac-tickets.json
│   ├── hg-changesets.json
│   └── wiki-pages.json
├── data/                 # export output + per-object maps
├── state/migration.yaml  # phase status/counters/errors
├── reports/              # preflight.md, validation.md
└── work/                 # scratch
```

`mappings/users.yaml` is the source of truth for identities. `state/` and
`data/*.json` are the source of truth for resume.

## Development

- Go 1.24. Build: `go build ./...`; tests: `go test ./...`; lint:
  `golangci-lint run ./...`.
- Keep the package boundaries: `pkg/gitlab`/`pkg/trac` are transports,
  `internal/importer` is the deterministic import path, `internal/validate`
  compares source and target.
- Add a phase by extending `pkg/gitlab`, `internal/importer`, the
  `migrate` command, and `internal/validate`. Return `importer.Result` and honor
  `config.DryRun`.
- Per-object mapping writes are atomic and are the resume mechanism. A mapping
  write failure is fatal; other object failures are recorded and skipped.
- Tests are table-driven where possible. No new dependencies without a reason.
- `docs/` is user-facing documentation. `AGENTS.md` and `skills/` are
  agent-facing. Keep the split.

## Where to look

- `docs/index.md` — user documentation.
- `skills/README.md` — skill index and status.
- `internal/` — engine implementation.
- `projects/` — local migration workspaces (never committed).
