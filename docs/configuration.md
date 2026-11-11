# Configuration

Configuration lives in `projects/<name>/migration.yaml`. Paths inside it are
resolved relative to the project directory.

## Environment variables

Secrets and connection details can come from `.env` in the project directory or
from the process environment. Existing environment variables win over `.env`.

| Variable | Purpose |
|---|---|
| `GITLAB_URL` / `GITLAB_HOST` | GitLab base URL; `https://` is added if missing |
| `GITLAB_TOKEN` | Personal access token with `api` scope |
| `GITLAB_PROJECT` | Project path, e.g. `group/project` |
| `GITLAB_PROJECT_ID` | Numeric project ID (alternative to path) |
| `TRAC_URL` | Trac base URL, e.g. `https://trac.example.com/my-project` |
| `TRAC_USER` | Trac XML-RPC username |
| `TRAC_PASSWORD` | Trac XML-RPC password |
| `TRAC_HTACCESS_USER` / `TRAC_HTACCESS_PASSWORD` | Optional htaccess gate in front of Trac; when set, this pair goes on the wire instead of `TRAC_USER`/`TRAC_PASSWORD` |
| `GITLAB_HTACCESS_USER` / `GITLAB_HTACCESS_PASSWORD` | Optional htaccess gate in front of GitLab; composes with token auth |

Never put tokens or passwords in `migration.yaml`.

## `migration.yaml`

```yaml
trac:
  base_url: https://trac.example.com
  rpc_path: /xmlrpc

gitlab:
  base_url: https://gitlab.example.com
  api_path: /api/v4
  project: group/project

export_options:
  include_wiki: true
  include_attachments: true
  include_ticket_history: true
  include_closed_tickets: true
  include_ticket_fields: true
  include_users: true
  export_dir: data
  wiki_exclude: []          # page names or globs to skip, e.g. [TracGuide, "Trac*"]
  wiki_exclude_authors: []  # skip pages whose latest author matches, e.g. [trac]

import_options:
  import_issues: true
  import_milestones: true
  import_notes: true
  import_attachments: true
  import_wiki: true
  users_file: mappings/users.yaml
  create_users: false

references:
  enabled: true
  tickets_file: mappings/trac-tickets.json
  changesets_file: mappings/hg-changesets.json
  wiki_file: mappings/wiki-pages.json

labels:
  enabled: true
  prefix: trac
  create_missing: true
  fields: [type, priority, resolution, component, version, severity, status]
```

## Sections

| Section | Meaning |
|---|---|
| `trac` | Trac base URL and XML-RPC path |
| `gitlab` | GitLab base URL/API path and project (path or ID) |
| `export_options` | What `trac2gitlab export` pulls from Trac |
| `import_options` | Which phases `trac2gitlab migrate` runs, and the identity mapping |
| `references` | Ticket/changeset/wiki reference maps |
| `labels` | Trac field-to-label mapping |

### Skipping wiki pages

Trac ships its own documentation as wiki pages (the TracGuide set), and an
untouched page stays authored by Trac's system account. To keep them out of the
migration, exclude them at export:

```yaml
export_options:
  wiki_exclude_authors: [trac]   # pages whose latest author is the system user
  wiki_exclude: ["Trac*"]        # names or globs (path.Match)
```

`wiki_exclude_authors` matches the author of the **latest** revision, so a
default-named page that a human later edited (e.g. `WikiStart`) is kept. Both
options default to empty, which migrates every page. Excluded pages are logged
and left out of `data/wiki`; references to them stay unchanged.

## Identity mapping

`mappings/users.yaml` maps Trac/HG authors to GitLab usernames:

```yaml
users:
  john-doe:
    trac_username: jdoe
    trac_display_name: John Doe
    gitlab: jdoe
    confidence: manual
  jane-doe:
    trac_username: jane
    action: unresolved
    confidence: manual
```

Any of `trac_username`, `trac_display_name`, `trac_email`, or `hg_author` can
match an author string. Unresolved authors become an explicit footer and are
skipped by the author patch pass.

Validate the mapping with:

```bash
trac2gitlab users validate --project-dir projects/my-project --check-gitlab
```

## Reference maps

- `mappings/trac-tickets.json`: `{"tickets": {"1": 3}}`
- `mappings/hg-changesets.json`: `{"changesets": {"<hg-node>": "<git-sha>"}}`
- `mappings/wiki-pages.json`: `{"pages": {"Trac Page": "gitlab-slug"}}` (optional)

Missing files are allowed. Ticket references then remain unchanged and
changeset references are left as-is. `hg-changesets.json` is produced outside
trac2gitlab (from an hg→git conversion); the engine only consumes it, and
changeset references inside notes cannot be corrected after the first import.

## Labels

`type=defect` becomes `trac:type:defect`. Existing labels are not removed;
missing labels are added on rerun. Set `labels.enabled: false` to disable the
mapping.
