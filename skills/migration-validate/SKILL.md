# migration-validate

## 1. Purpose

Compare the Trac export with the GitLab target and produce the final verdict.
Read-only against both systems.

## 2. Preconditions

- `trac2gitlab migrate --execute` has completed.
- `trac2gitlab patch` has been applied if historical authors matter (otherwise
  author checks will warn).
- `data/` export and mapping files are present.

## 3. Workflow

1. **Run validation.**
   ```bash
   trac2gitlab validate --project-dir projects/<name>
   ```
   It writes `reports/validation.md` and exits non-zero on `FAIL`.

2. **Present the summary.**

   | Area | Expected | Found | Matched | Failed |
   |---|---|---|---|---|
   | Issues | | | | |
   | Notes | | | | |
   | Attachments | | | | |
   | Wiki | | | | |

   Then list errors and warnings.

3. **Interpret.**
   - **Errors** — missing issues/notes/attachments/wiki pages, or missing
     objects referenced by mappings. `FAIL`.
   - **Warnings** — timestamp drift, author not patched, missing labels,
     unlinked attachments. Decide with the user whether each is acceptable.
   - `PASS` / `PASS WITH WARNINGS` / `FAIL` is the final status.

4. **Remediate and rerun.**
   - Missing content: fix the cause and rerun `migrate` (idempotent).
   - Author warnings: rerun `patch`, then `validate`.
   - Timestamp drift: compare the Trac value with the GitLab value; if it is a
     real limitation, document it; do not silently accept a migration-date
     fallback.

5. **Report.** Summarise:
   - verdict
   - counts per area
   - unresolved identities and any footer-only authors
   - warnings accepted by the user
   - remaining manual work (attachments, wiki history, etc.)

## 4. Outputs

- `reports/validation.md`
- Final verdict and summary

## 5. Safety

- Strictly read-only. No target writes.
- Do not hide warnings; the user decides whether to accept them.
- Do not claim `PASS` when the report says `FAIL` or when the command exits
  non-zero.

## 6. Failure / retry

- Validation is safe to rerun after any fix.
- If the export is stale, rerun `trac2gitlab export` before validating.
- If the target was partially migrated, rerun `migrate` first; completed objects
  are skipped.
- Author warnings after `patch` usually mean an unresolved/incorrect mapping;
  fix `mappings/users.yaml` and regenerate the patch.
