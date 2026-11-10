# trac-to-gitlab

## 1. Purpose

Orchestrate the Trac content import: export, dry-run, explicit confirmation,
execute. This skill does not transform content or call GitLab directly; the Go
engine does that.

## 2. Preconditions

- `reports/preflight.md` exists and is `READY` or `READY WITH WARNINGS`
  accepted by the user.
- `mappings/users.yaml` passed `trac2gitlab users validate --check-gitlab`.
- `projects/<name>/migration.yaml` and `.env` are present.
- Optional: `mappings/hg-changesets.json` is present if changeset references
  should point at Git commits. It is produced outside this tool (an hg→git
  conversion); trac2gitlab only consumes it.
- The target project is fresh, or the user explicitly accepted the offset.

Do not start if preflight is `BLOCKED` or `NOT READY`.

## 3. Workflow

1. **Refresh the export.**
   ```bash
   trac2gitlab export --project-dir projects/<name>
   ```
   Read-only against Trac. If the source changed since the last export, this
   keeps the import consistent with it. It exits non-zero if any phase fails;
   fix the cause and rerun. Changing `export_options` (for example
   `wiki_exclude`) takes effect only on re-export; the wiki phase logs
   `excluded=N`.

2. **Dry run.**
   ```bash
   trac2gitlab migrate --project-dir projects/<name>
   ```
   `migrate` defaults to dry-run. It writes `state/migration.yaml` with
   per-phase counts but no GitLab objects.

3. **Review the plan.** Run:
   ```bash
   trac2gitlab status --project-dir projects/<name> --json
   ```
   Present the per-phase counts and first errors. A dry-run with failures must
   be fixed before executing.

4. **Confirm.** Ask explicitly: write these counted objects to GitLab? Do not
   interpret silence as consent. If the user hesitates, stay in dry-run.

5. **Execute.**
   ```bash
   trac2gitlab migrate --project-dir projects/<name> --execute
   ```

6. **Check the result.** Read `state/migration.yaml` again. If the command
   exits non-zero, present the failed phase and its error list. Fix the cause
   and rerun the same command; the engine skips completed objects and retries
   failed ones.

7. **Hand off.**
   - Historical authors: `migration-patch`.
   - Final comparison: `migration-validate`.

## 4. Outputs

- `data/` export
- `state/migration.yaml`
- GitLab issues, notes, attachments, wiki (only with `--execute`)

## 5. Safety

- No target mutation before preflight passes and the user confirms.
- `--execute` is explicit; never add a force/ignore flag.
- Historical timestamps and authors are mandatory. If a timestamp cannot be
  preserved, the phase records a failure; do not rewrite it to the migration
  date.
- Never silently attribute content to the migration account.
- Never silently drop unsupported content; warnings come from the engine.

## 6. Failure / retry

- Per-object failures are recorded; the run continues where safe.
- Mapping-file write failures are fatal (they protect against duplicates).
- Rerun the same command after fixing; completed notes/uploads/wiki pages are
  skipped via the exported object maps.
- If issue creation fails with an IID mismatch, stop: the token lacks
  admin/Owner rights.
