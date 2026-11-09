# Getting started

This tutorial takes you from an empty project directory to a validated dry-run.

## Requirements

- Go 1.24 or later
- A Trac instance with the XML-RPC plugin enabled
- A GitLab project and a personal access token with `api` scope
- For historical authors: a GitLab admin or project Owner token, and shell
  access to the GitLab host for the final `patch` step

Use a **fresh, empty** GitLab project. GitLab issue IIDs are sequential and
never reused; pre-existing issues break 1:1 Trac-to-GitLab ID parity.

## Build

```bash
git clone https://github.com/bnidev/trac2gitlab.git
cd trac2gitlab
go build -o trac2gitlab ./cmd/trac2gitlab
```

## Create a migration project

```bash
./trac2gitlab init --project-dir projects/my-project
```

This creates `projects/my-project/migration.yaml`. Edit it:

```yaml
trac:
  base_url: https://trac.example.com
  rpc_path: /xmlrpc

gitlab:
  base_url: https://gitlab.example.com
  api_path: /api/v4
  project: group/my-project
```

## Add credentials

Create `projects/my-project/.env`:

```text
GITLAB_TOKEN=...
TRAC_USER=...
TRAC_PASSWORD=...
```

`GITLAB_URL`/`GITLAB_HOST` and `GITLAB_PROJECT` may also be set in `.env` or
the environment. Existing environment variables win over `.env`. Secrets never
go into `migration.yaml`.

## Run the readiness check

```bash
./trac2gitlab preflight --project-dir projects/my-project
```

This is read-only. It writes `projects/my-project/reports/preflight.md` and
blocks on a role below Owner/Administrator or an unreachable source.

## Export from Trac

```bash
./trac2gitlab export --project-dir projects/my-project
```

This writes tickets, comments, wiki pages, and attachments under
`projects/my-project/data/`.

## Propose identities

```bash
./trac2gitlab discover --project-dir projects/my-project \
  --write-users mappings/users.yaml
```

Review the authors and map each to a GitLab username. Keep unknowns as
`action: unresolved`. Validate with:

```bash
./trac2gitlab users validate --project-dir projects/my-project --check-gitlab
```

## Review the import plan

```bash
./trac2gitlab migrate --project-dir projects/my-project
```

`migrate` defaults to `--dry-run`: it reads GitLab but writes nothing. Review
the log and `projects/my-project/state/migration.yaml`.

## Write to GitLab

```bash
./trac2gitlab migrate --project-dir projects/my-project --execute
```

## Restore historical authors

```bash
./trac2gitlab patch --project-dir projects/my-project
```

This prints a dry-run block. Paste it into the GitLab host shell (see
[Migration flow](migration-flow.md#author-patch)). After reviewing, emit the
applying block:

```bash
./trac2gitlab patch --project-dir projects/my-project --apply
```

## Validate

```bash
./trac2gitlab validate --project-dir projects/my-project
```

The report is written to `projects/my-project/reports/validation.md`. The
command exits non-zero on `FAIL`.

## Next

- [Configuration](configuration.md)
- [Limitations](limitations.md)
