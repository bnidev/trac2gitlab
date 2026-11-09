# Trac2GitLab

A tool to synchronize tickets, wikis and related data from a Trac instance (via XML-RPC) into a GitLab project.

[![CI](https://github.com/bnidev/trac2gitlab/actions/workflows/ci.yaml/badge.svg)](https://github.com/bnidev/trac2gitlab/actions/workflows/ci.yaml)


## Current Features

### Exporter

- Export tickets from Trac as JSON files (including content history and comments)
- Export content of wiki pages from Trac as markdown files (including history)
- Export metadata of wiki pages from Trac as JSON files (including history)
- Export milestones as JSON files
- Concurrent export operations for faster migration (speed might be limited by Trac XML-RPC)
- Download attachments
- Configurable via YAML

### Importer

- Import milestones into GitLab projects (updates if already exist and content differs)
- Import issues into GitLab projects (updates if already exist and content differs)
- Import ticket comments with original timestamps; note authors are recorded for the author patch pass
- Upload ticket attachments and link them from issue descriptions
- Import wiki pages and attachments (latest version; original author/date recorded in a footer)
- Rewrite ticket IDs and changeset references using explicit mapping files
- Map Trac fields to GitLab labels (`trac:type:defect`, `trac:priority:major`, ...)

## Planned Features

### Importer

- Preserve ticket history in GitLab issues (not supported by the GitLab API)
- Map Trac fields to GitLab labels
- Preserve wiki page history in GitLab wikis (not supported by the GitLab API)

## Compatibility

This package is compatible with:

- **TracXMLRPC version:** 1.1.9
- **Trac core version:** 1.2.3

Tested against the plugin installed from: https://trac-hacks.org/wiki/XmlRpcPlugin

If you're using another version, some methods may not be available or may behave differently.

## Requirements

- Go 1.24 or later
- Access to a Trac instance with XML-RPC enabled
- GitLab project with API access token

## Running the tool (from source)

1. Clone the repository
2. Run `make install` to install dependencies
3. Create a project directory:
   ```bash
   trac2gitlab init --project-dir projects/my-project
   ```
4. Edit `projects/my-project/migration.yaml` and create
   `projects/my-project/.env` (see `.env.example`). Credentials are read from
   the environment or `.env`, never from `migration.yaml`:
   ```text
   GITLAB_URL=https://gitlab.example.com
   GITLAB_TOKEN=...
   GITLAB_PROJECT=group/project
   TRAC_URL=https://trac.example.com
   TRAC_USER=...
   TRAC_PASSWORD=...
   ```
5. Run commands against the project:
   ```bash
   trac2gitlab export  --project-dir projects/my-project
   trac2gitlab migrate --project-dir projects/my-project            # dry-run
   trac2gitlab migrate --project-dir projects/my-project --execute  # write
   ```

**Trac Exporter:**

```bash
trac2gitlab export --project-dir projects/my-project
```

**GitLab Importer:**

```bash
trac2gitlab migrate --project-dir projects/my-project --execute
```

`migrate` defaults to `--dry-run`; per-phase progress is written to
`projects/<name>/state/migration.yaml`.

## Reference rewriting

Ticket and changeset references are rewritten during import using explicit maps:

```yaml
references:
  enabled: true
  tickets_file: mappings/trac-tickets.json
  changesets_file: mappings/hg-changesets.json
```

- `trac-tickets.json`: `{"tickets": {"1": 3}}` (Trac ID → GitLab IID)
- `hg-changesets.json`: `{"changesets": {"<hg-node>": "<git-sha>"}}`

Missing files are allowed: ticket references stay unchanged, and changeset
references are left as-is. Fenced code blocks are not rewritten.

## Labels

Trac fields become GitLab labels:

```yaml
labels:
  enabled: true
  prefix: trac
  create_missing: true
  fields: [type, priority, resolution, component, version, severity, status]
```

For example `type=defect` becomes `trac:type:defect`. Existing labels are not
removed; missing labels are added on rerun.

## Restoring authors

The importer creates issues/notes with the admin token, because GitLab only lets
admins/owners set historical timestamps and IIDs. To restore the original
authors, emit the patch block and run it on the GitLab host:

```bash
# dry run: prints what would change
trac2gitlab patch

# after reviewing the dry run, emit an applying block
trac2gitlab patch --apply
```

Paste the emitted block into the GitLab host shell. It writes a temporary Ruby
file inside the container, runs `gitlab-rails runner`, and deletes the file.
Use `--container <name>` if the container is not called `gitlab`, or
`--omnibus` for a host install. The block updates `issues.author_id` and
`notes.author_id` only; unmapped authors are reported and skipped.

Check the `Makefile` for more commands and options.
