# Development

## Layout

```text
cmd/trac2gitlab/       CLI entry point
internal/app/          shared AppContext, project ID/path resolution
internal/cli/          Cobra commands
internal/config/       migration.yaml + .env + path resolution
internal/exporter/     Trac → data/
internal/importer/     data/ → GitLab, mappings, state results
internal/identity/     users.yaml + GitLab user resolution
internal/labels/       Trac field → GitLab label mapping
internal/references/   ticket/changeset/wiki reference rewriting
internal/state/        state/migration.yaml
internal/validate/     export vs target validation
pkg/gitlab/            GitLab REST adapter
pkg/trac/              Trac XML-RPC client
pkg/xmlrpc/            XML-RPC transport
skills/                agent-driven workflow (not user docs)
docs/                  user documentation
```

Projects under `projects/` are local only and gitignored.

## Build, test, lint

```bash
go build ./...
go test ./...
golangci-lint run ./...
```

The Makefile provides the same commands plus `make build`.

## Adding an import phase

1. Add the GitLab REST call to `pkg/gitlab/`.
2. Add `internal/importer/<phase>.go` returning `importer.Result`.
3. Honor `config.DryRun` and record per-object errors.
4. Add the phase to `internal/cli/migrate.go`.
5. Write a mapping file for idempotency if the phase creates objects.
6. Add validation checks in `internal/validate/`.
7. Add tests and update `docs/`.

## Configuration

`migration.yaml` is loaded by `internal/config`; relative paths resolve against
the project directory. Secrets come from `.env` or the environment only.

## Documentation

- `docs/` is the public user documentation.
- The agent-driven workflow is documented separately from these user pages.
- Keep user docs free of agent-only orchestration details.

## Agent workflow

The agent-driven migration workflow is maintained separately from these user
docs. Deterministic transformation lives in the Go engine, never in the skill
prose.
