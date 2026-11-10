# migration-patch

## 1. Purpose

Restore historical issue and note authors on the GitLab host. GitLab's API
cannot set an author, so this is a one-time server-side step using
`gitlab-rails runner`.

## 2. Preconditions

- `trac2gitlab migrate --execute` has completed (or partially completed; rerun
  is idempotent).
- `data/trac-notes.json` and the exported tickets exist.
- `mappings/users.yaml` is approved.
- Preflight confirmed shell/console access to the GitLab host.
- The user has an admin/Owner token for the API and shell access for the runner.

## 3. Workflow

1. **Emit the dry run.**
   ```bash
   trac2gitlab patch --project-dir projects/<name>
   ```
   This prints one shell block. It writes `/tmp/patch_authors.rb` inside the
   GitLab container, runs `gitlab-rails runner`, and deletes the file. No
   changes are made because `DRY_RUN = true`.

2. **Hand the block to the user.** They run it on the GitLab host. If the
   container is not `gitlab`, use:
   ```bash
   trac2gitlab patch --project-dir projects/<name> --container <name>
   ```
   For an Omnibus host install, use `--omnibus` instead of `docker exec`.

3. **Review the dry run.** The output lists every `issue #N` / `note #N` that
   would change and any rows that are already correct or missing. If rows are
   skipped as unresolved, confirm that this matches the approved identity
   mapping.

4. **Apply.**
   ```bash
   trac2gitlab patch --project-dir projects/<name> --apply
   ```
   Give the user the new block to run on the host. The engine does not execute
   Rails itself.

5. **Recheck.**
   ```bash
   trac2gitlab validate --project-dir projects/<name>
   ```
   Author warnings should clear. Remaining author warnings mean a mapping or
   patch failure; report them; do not silently accept them.

## 4. Outputs

- Updated `issues.author_id` / `notes.author_id` in GitLab
- A validation report showing author status

## 5. Safety

- Only `author_id` is changed. `closed_at` is set by the API during import.
- The script is idempotent: already-correct rows are skipped.
- Never run `gitlab-rails` automatically; the user runs the block on the host.
- Do not paste the block into a shared channel; it contains only IDs, but still
  treat it as operational output.
- Unresolved authors are skipped and keep their `Originally by ...` footer.
- Only issues and notes are reassigned. System notes (label changes,
  cross-reference "mentioned in …" notes, and the description update that
  appends the attachments section) keep the migration account, because they are
  actions the migration performed. See `docs/limitations.md`.

## 6. Failure / retry

- Container name wrong or `gitlab-rails` missing: fix `--container`/`--omnibus`
  and rerun. Nothing was changed.
- Partial apply: rerun the `--apply` block; already-correct rows are skipped.
- Author still wrong after apply: check `mappings/users.yaml`, rerun
  `users validate --check-gitlab`, then regenerate the patch.
- Wiki history/authors are not part of this step; wiki author/date lives in the
  page footer.
