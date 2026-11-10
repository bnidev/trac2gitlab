# Limitations

## GitLab API

- **Historical timestamps** require an admin or project Owner token.
  `created_at` and explicit `iid` are silently dropped for normal users.
- **Authors** cannot be set through the REST API. `trac2gitlab patch` sets
  `issues.author_id` / `notes.author_id` with `gitlab-rails runner` on the
  GitLab host.
- **`closed_at`** is set by passing `updated_at` on close. It therefore follows
  the Trac `TimeChanged` value. If a ticket was closed at a different time from
  its last change, the close date will follow the last change.
- **Wiki history** is not preserved. Only the latest Trac page version is
  imported; the original author/date/version are recorded in a footer.
- **Attachment metadata** is not stored by GitLab. The importer writes the
  original uploader/date into the issue or wiki content.
- **System notes** (label changes, cross-reference “mentioned in …” notes, and
  the description update that appends the attachments section) are attributed
  to the **migration account**, and may carry the migration timestamp or a
  backdated one. They are actions the migration performed, not historical
  users, so they are deliberately left as the migration account rather than
  reassigned. Issue and note authors *are* restored by `trac2gitlab patch`.
- **Description history** from Trac is exported but not yet imported as notes.
- **Keywords/custom fields** are not mapped to labels (standard fields are).

## Reference rewriting

- Fenced code blocks are protected; inline code is not.
- Changeset references shorter than a full hash resolve to the first matching
  prefix. Use full hashes in the mapping if collisions matter.
- Wiki slugs are approximated. Use `mappings/wiki-pages.json` for overrides.
- Unknown ticket/changeset/wiki references are left unchanged.

## Target project

- GitLab issue IIDs are sequential and never reused. Exact
  `Trac #N → GitLab #N` parity requires a **fresh empty project**.
- Pre-existing issues, including test/probe issues, shift all later IIDs.

## Source and transport

- Trac must expose XML-RPC at `<trac-url>/xmlrpc`.
- The engine does not scrape HTML and does not use a Trac MCP server. A local
  MCP server may still be useful for ad-hoc agent inspection.
- Trac author strings may be usernames, display names, or emails. Add the right
  aliases to `mappings/users.yaml`; unresolved authors get a footer.

## Failure behavior

- Per-object failures are recorded and skipped; the phase is marked failed and
  the command exits non-zero.
- Mapping-file write failures are fatal: they protect against duplicate notes
  or uploads on rerun.
- A partial run can leave the target in a mixed state. Inspect
  `state/migration.yaml`, fix the cause, and rerun; completed objects are
  skipped.
