# CLI reference

All commands accept the global flag:

```text
--project-dir string   migration project directory (default ".")
```

## `init`

```bash
trac2gitlab init --project-dir projects/my-project
```

Creates `migration.yaml` with defaults. Does not overwrite an existing config.

## `preflight`

```bash
trac2gitlab preflight --project-dir projects/my-project
```

Read-only readiness check. Writes `reports/preflight.md` with
`STATUS: READY | READY WITH WARNINGS | NOT READY | BLOCKED`.

Checks Trac XML-RPC, GitLab project/role, historical-timestamp capability,
target issue count, and unresolved identities. Exits non-zero on `BLOCKED` or
`NOT READY`.

## `export`

```bash
trac2gitlab export --project-dir projects/my-project
```

Reads Trac over XML-RPC and writes tickets, comments, wiki pages, and
attachments under the configured export directory (default `data/`).

## `discover`

```bash
trac2gitlab discover --project-dir projects/my-project
trac2gitlab discover --project-dir projects/my-project --write-users mappings/users.yaml
```

Summarises the export: ticket/comment/wiki/attachment counts and every author.
`--write-users` writes a `users.yaml` skeleton with all authors unresolved; it
refuses to overwrite an existing file.

## `migrate`

```bash
trac2gitlab migrate --project-dir projects/my-project            # dry-run
trac2gitlab migrate --project-dir projects/my-project --execute  # write
```

| Flag | Meaning |
|---|---|
| `--dry-run` | Read-only preview; default when neither flag is given |
| `--execute` | Write to GitLab |

Phases: milestones → issues → notes → attachments → wiki. Progress is written
to `state/migration.yaml`. `--dry-run` and `--execute` are mutually exclusive.

## `patch`

```bash
trac2gitlab patch --project-dir projects/my-project
trac2gitlab patch --project-dir projects/my-project --apply
```

Emits a copy-paste shell block that sets `issues.author_id` and
`notes.author_id` via `gitlab-rails runner`. Dry-run by default; `--apply`
emits the applying version.

| Flag | Meaning |
|---|---|
| `--apply` | Set `DRY_RUN = false` in the emitted script |
| `--container` | GitLab container name (default `gitlab`) |
| `--omnibus` | Emit commands for a host install (no `docker exec`) |

## `validate`

```bash
trac2gitlab validate --project-dir projects/my-project
```

Compares the export with GitLab and writes `reports/validation.md`. Exits
non-zero on `FAIL`.

## `users validate`

```bash
trac2gitlab users validate --project-dir projects/my-project
trac2gitlab users validate --project-dir projects/my-project --check-gitlab
```

Validates `mappings/users.yaml`. `--check-gitlab` resolves usernames to numeric
IDs. Exits non-zero if any user is unresolved.

| Flag | Meaning |
|---|---|
| `--users` | Path to `users.yaml`; defaults to `import_options.users_file` |
| `--check-gitlab` | Resolve GitLab usernames to IDs |

## `version`

```bash
trac2gitlab version
```

Prints the build version.
