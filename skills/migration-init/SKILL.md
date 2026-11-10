# migration-init

## 1. Purpose

Create a migration project, write `migration.yaml`, export the Trac source, and
produce a proposed `mappings/users.yaml`. This skill is read-only against Trac
and GitLab.

## 2. Inputs

- Project name (required; `^[a-z0-9][a-z0-9-]*$`).
- Trac URL (required).
- GitLab project path, e.g. `group/project` (required).
- Mercurial clone URL(s) (optional here; used by `hg-to-gitlab`).

Never ask the user to paste secrets into chat. They go into
`projects/<name>/.env`.

## 3. Workflow

1. **Scaffold.** Run:
   ```bash
   trac2gitlab init --project-dir projects/<name>
   ```
   Do not overwrite an existing project; stop and ask.

2. **Configure.** Edit `projects/<name>/migration.yaml`:
   ```yaml
   trac:
     base_url: <trac-url>
     rpc_path: /xmlrpc
   gitlab:
     base_url: <gitlab-url>
     project: <group/project>
   mercurial:
     repositories:
       - name: main
         clone_url: <hg-clone-url>
   ```
   The engine ignores `mercurial`; `hg-to-gitlab` uses it.

   By default every Trac wiki page is migrated, including Trac's own shipped
   documentation (the TracGuide set). Those pages stay authored by the Trac
   system user, so to keep them out add:
   ```yaml
   export_options:
     wiki_exclude_authors: [trac]   # pages whose latest author is the system user
     wiki_exclude: ["Trac*"]        # names or globs (path.Match)
   ```
   Empty config migrates everything. See `docs/configuration.md`.

3. **Credentials.** Create `projects/<name>/.env` from `.env.example` and tell
   the user to fill in:
   ```text
   GITLAB_TOKEN=...
   TRAC_USER=...
   TRAC_PASSWORD=...
   ```
   `GITLAB_URL`/`GITLAB_HOST` and `GITLAB_PROJECT` may live in `.env` or the
   environment.

4. **Export.** Run:
   ```bash
   trac2gitlab export --project-dir projects/<name>
   ```
   This is read-only against Trac. If it fails on 401 or network, stop and ask
   the user to fix `.env`/VPN; do not fall back to HTML scraping or MCP.

5. **Discover authors.** Run:
   ```bash
   trac2gitlab discover --project-dir projects/<name> \
     --write-users mappings/users.yaml
   ```
   This prints ticket/comment/wiki/attachment counts and every author. If
   `mappings/users.yaml` already exists, the command refuses to overwrite;
   merge manually instead.

6. **Offer to exclude Trac's own documentation.** Trac ships its manual as wiki
   pages created by the system account at install time; they are almost never
   wanted in the target. The `discover` output lists that author — commonly
   `trac`, but sometimes `admin` or the installer's name — and its pages are all
   version 1 with no edit comment. If such pages exist, ask the user whether to
   keep them out, and if so add to `migration.yaml`:
   ```yaml
   export_options:
     wiki_exclude_authors: [trac]
   ```
   Never exclude silently and never guess the username: confirm it with the user
   or from a page's metadata. Leaving it empty migrates every page.

7. **Propose identities.** For each author, ask the user for the GitLab
   username. Write the result to `mappings/users.yaml`:
   ```yaml
   users:
     john-doe:
       trac_username: jdoe
       gitlab: jdoe
       confidence: manual
     unknown-user:
       trac_username: mystery
       action: unresolved
       confidence: manual
   ```
   Match on any of `trac_username`, `trac_display_name`, `trac_email`,
   `hg_author`. Never invent a mapping.

8. **Validate identities.** Run:
   ```bash
   trac2gitlab users validate --project-dir projects/<name> --check-gitlab
   ```
   If the GitLab token is not available yet, run without `--check-gitlab` and
   record that ID resolution is pending.

9. **Report.** Summarise counts, unresolved users, and the next step
   (`migration-preflight`). Record warnings; do not hide them.

## 4. Outputs

- `projects/<name>/migration.yaml`
- `projects/<name>/.env` (user-filled, gitignored)
- `projects/<name>/data/` (export)
- `projects/<name>/mappings/users.yaml` (proposed)

## 5. Safety

- No writes to Trac or GitLab.
- No secrets in `migration.yaml`, mappings, reports, or chat.
- Do not overwrite an existing project or `users.yaml` silently.
- Do not use Trac MCP or HTML scraping; the engine uses XML-RPC.

## 6. Failure / retry

- Missing credentials: stop, list the missing `.env` keys, let the user fix and
  rerun.
- Export partial failure: report the failing phase/ticket and rerun after the
  cause is fixed.
- Discovery failure: report the missing export files; do not invent counts.
- Init is idempotent: rerunning does not destroy an existing config or approved
  mapping.
