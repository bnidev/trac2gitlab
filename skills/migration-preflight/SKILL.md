# migration-preflight

## 1. Purpose

Gate the migration before any target mutation. Read-only against Trac and
GitLab. Produces `reports/preflight.md` and a verdict.

Historical timestamps and authors are mandatory. If the current credentials
cannot preserve them, the migration must not start. There is no
`--force`/`--ignore-permissions` bypass.

## 2. Required state

- `projects/<name>/migration.yaml`
- `projects/<name>/.env` with `GITLAB_TOKEN`, `TRAC_USER`, `TRAC_PASSWORD`
- `projects/<name>/mappings/users.yaml` (proposed; unresolved entries allowed
  only if the user explicitly accepts footers)

## 3. Workflow

1. **Check files.** If `migration.yaml`, `.env`, or `users.yaml` is missing,
   stop and send the user back to `migration-init`.

2. **Run the engine check.**
   ```bash
   trac2gitlab preflight --project-dir projects/<name>
   ```
   It writes `reports/preflight.md` and exits non-zero on `BLOCKED` or
   `NOT READY`.

3. **Read the report.** Present the table. The verdict is one of:

   | Verdict | Meaning | Action |
   |---|---|---|
   | `READY` | all checks pass | proceed |
   | `READY WITH WARNINGS` | non-empty target, unresolved users, plugin version, etc. | list every warning and get explicit acceptance |
   | `NOT READY` | Trac unreachable/methods missing, GitLab unreachable, identity file missing | stop; fix and rerun |
   | `BLOCKED` | current role cannot set historical timestamps | stop; no target writes |

4. **Permission gate.**
   - Required: GitLab project **Owner** or instance **Administrator**.
   - **Maintainer is insufficient** and must produce `BLOCKED`.
   - A live timestamp probe is deliberately not used: it would consume a
     sequential issue IID and break Trac-to-GitLab parity. Capability is
     inferred from role.
   - Historical authors additionally require shell/console access to the GitLab
     host for the one-time `trac2gitlab patch` step. Ask the user to confirm
     this access; the engine cannot check it.

5. **Identity gate.**
   ```bash
   trac2gitlab users validate --project-dir projects/<name> --check-gitlab
   ```
   - Unresolved users are allowed only if the user explicitly accepts the
     `Originally by ...` footer behavior.
   - Never invent a GitLab user.
   - Record the decision.

6. **Target gate.**
   - Confirm the target project is **fresh and empty** if exact
     `Trac #N → GitLab #N` parity matters.
   - If issues already exist, record the offset as a warning and get explicit
     confirmation before importing.

7. **Report and hand off.** Summarise: verdict, role, warnings, unresolved users,
   target issue count, and next step (`trac-to-gitlab`). Do not mutate GitLab.

## 4. Outputs

- `reports/preflight.md`
- A short summary to the user

## 5. Safety

- Strictly read-only. No issues, notes, uploads, wiki pages, or repository
  writes.
- No test issue/probe that consumes an IID.
- No secrets in the report.
- Do not accept `--force` or ignore a `BLOCKED` verdict.

## 6. Failure / retry

- Trac/GitLab unreachable: report the error, let the user fix credentials/VPN,
  rerun preflight.
- Role insufficient: the user must obtain Owner or Administrator access, then
  rerun. Rerunning preflight alone does not grant access.
- Identity gaps: resolve in `users.yaml` and rerun `users validate`.
- Preflight is safe to rerun; it writes only the report.
