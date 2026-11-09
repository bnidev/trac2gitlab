# Skills

Agent orchestration contracts for the Trac2GitLab migration. The Go engine does
the transformation; these skills decide *when* to run it, *what* to ask the
user, and *how* to report results.

User documentation is in `../docs/`. Agent development rules are in
`../AGENTS.md`.

## Status

| Skill | Status | Purpose |
|---|---|---|
| `migration-init` | available | Create the project, write `migration.yaml`, propose identities |
| `migration-preflight` | available | Gate on access, timestamp capability, identities, target state |
| `trac-to-gitlab` | available | Export → dry-run → execute |
| `migration-patch` | available | Author patch handoff and recheck |
| `migration-validate` | planned | Run validation and report |
| `hg-to-gitlab` | separate domain | Mercurial → Git repository migration (not in the Go engine) |

“Planned” means the file has not landed in this repository yet. Each skill lands
in its own PR.

## Flow

```text
migration-init
  → migration-preflight
  → trac-to-gitlab (export, dry-run, execute)
  → migration-patch
  → migration-validate
```

`hg-to-gitlab` runs before `trac-to-gitlab` when changeset references must point
at real commits, but it is a separate domain.

## Conventions

- Every command takes `--project-dir projects/<name>`.
- `migrate` defaults to dry-run. `--execute` requires explicit user
  confirmation.
- The preflight verdict gates all target mutation.
- Authors are restored on the GitLab host via `trac2gitlab patch`; the agent
  never runs `gitlab-rails` itself.
- Reports go to `projects/<name>/reports/`; state goes to
  `projects/<name>/state/migration.yaml`.
- Skills do not contain transformation logic. If a conversion or import rule is
  missing, extend the Go engine.
