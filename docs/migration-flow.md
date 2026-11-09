# Migration flow

```text
init → preflight → export → migrate --dry-run → migrate --execute → patch → validate
```

## Project layout

```text
projects/<name>/
├── migration.yaml
├── .env                     # gitignored
├── mappings/
│   ├── users.yaml
│   ├── trac-tickets.json
│   ├── hg-changesets.json
│   └── wiki-pages.json
├── data/                    # export output
│   ├── tickets/
│   ├── wiki/
│   ├── trac-notes.json
│   ├── trac-attachments.json
│   └── trac-wiki-attachments.json
├── state/migration.yaml
├── reports/validation.md
└── work/
```

## Phases

`migrate` runs these phases in order, skipping disabled ones:

1. milestones
2. issues (title, description, state, created/closed time, labels, milestone)
3. notes (comments with original timestamps)
4. attachments (upload and link from issue descriptions)
5. wiki (latest page version; attachments; footer with original author/date)

`migrate` defaults to `--dry-run`. Pass `--execute` to write.

## State and resume

`state/migration.yaml` records overall status and per-phase counters:

```yaml
status: failed
dry_run: false
started_at: 2026-11-09T13:00:00Z
phases:
  issues:
    status: completed
    total: 11
    completed: 11
    failed: 0
  notes:
    status: failed
    total: 20
    completed: 17
    failed: 3
    errors:
      - "..."
```

Per-object mappings (`data/trac-notes.json`, `data/trac-attachments.json`,
`data/trac-wiki-attachments.json`) let reruns skip completed objects. Rerun the
same command to retry failed work.

## Author patch

GitLab's API cannot set an issue/note author. The importer creates them with the
admin/Owner token, then `patch` emits a script that sets `author_id` on the
GitLab host.

```bash
trac2gitlab patch --project-dir projects/my-project
```

Paste the output into the GitLab host shell. The block writes
`/tmp/patch_authors.rb` inside the container, runs `gitlab-rails runner`, and
deletes it. Use `--container <name>` if the container is not `gitlab`, or
`--omnibus` on a host install. When the dry-run looks right:

```bash
trac2gitlab patch --project-dir projects/my-project --apply
```

Unresolved authors are skipped and already have an explicit
`Originally by ...` footer.

## Validation

```bash
trac2gitlab validate --project-dir projects/my-project
```

The report covers issues, notes, attachments, wiki, labels, timestamps, and
authors. Missing objects are errors; author/timestamp drift is a warning.
