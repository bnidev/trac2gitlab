# Trac2GitLab

Migrate Trac tickets, comments, attachments, and wiki pages into a GitLab
project while preserving historical timestamps and, with one server-side patch
step, historical authors.

Trac data is read over **XML-RPC**. GitLab writes go through the **REST API**.

## Start here

- [Getting started](getting-started.md) — create a project and run a dry-run.
- [Configuration](configuration.md) — `migration.yaml` and environment variables.
- [CLI reference](cli.md) — every command and flag.
- [Migration flow](migration-flow.md) — phases, state, resume, author patch.
- [Limitations](limitations.md) — what GitLab cannot preserve.
- [Development](development.md) — repo layout, build, test, and how to add a phase.

## What it preserves

- Ticket title, description, creation/close time, state, milestones
- Ticket comments with their original timestamps
- Ticket and wiki attachments (uploaded and linked)
- Wiki pages (latest version; original author/date in a footer)
- Ticket, changeset, and wiki references
- Trac fields as GitLab labels (`trac:type:defect`, `trac:priority:major`, ...)
- Historical authors after the `trac2gitlab patch` step

## What it cannot preserve

- GitLab does not expose historical wiki page history.
- GitLab's API cannot set the author of an issue or note. The `patch` step does
  it on the GitLab host with `gitlab-rails runner`.
- `closed_at` follows the Trac `TimeChanged` value; see [Limitations](limitations.md).

## Agent-driven flow

The repository also includes an agent-driven migration workflow. It is
documented separately from these user pages and will live alongside the skills
used by the agent.
