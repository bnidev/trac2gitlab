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
changeset references are left as-is.

## Labels

`type=defect` becomes `trac:type:defect`. Existing labels are not removed;
missing labels are added on rerun. Set `labels.enabled: false` to disable the
mapping.
