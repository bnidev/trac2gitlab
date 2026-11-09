# Trac2GitLab

A tool to synchronize tickets, wikis and related data from a Trac instance (via XML-RPC) into a GitLab project.

[![CI](https://github.com/bnidev/trac2gitlab/actions/workflows/ci.yaml/badge.svg)](https://github.com/bnidev/trac2gitlab/actions/workflows/ci.yaml)


## Current Features

### Exporter

- Export tickets from Trac as JSON files (including content history and comments)
- Export content of wiki pages from Trac as markdown files (including history)
- Export metadata of wiki pages from Trac as JSON files (including history)
- Export milestones as JSON files
- Concurrent export operations for faster migration (speed might be limited by Trac XML-RPC)
- Download attachments
- Configurable via YAML

### Importer

- Import milestones into GitLab projects (updates if already exist and content differs)
- Import issues into GitLab projects (updates if already exist and content differs)
- Import ticket comments with original timestamps; note authors are recorded for the author patch pass
- Upload ticket attachments and link them from issue descriptions
- Import wiki pages (latest version; original author/date recorded in a footer)

## Planned Features

### Importer

- Preserve ticket history in GitLab issues (not supported by the GitLab API)
- Map Trac fields to GitLab labels
- Import wiki attachments
- Preserve wiki page history in GitLab wikis (not supported by the GitLab API)

## Compatibility

This package is compatible with:

- **TracXMLRPC version:** 1.1.9
- **Trac core version:** 1.2.3

Tested against the plugin installed from: https://trac-hacks.org/wiki/XmlRpcPlugin

If you're using another version, some methods may not be available or may behave differently.

## Requirements

- Go 1.23 or later
- Access to a Trac instance with XML-RPC enabled
- GitLab project with API access token

## Running the tool (from source)

1. Clone the repository
2. Run `make install` to install dependencies
3. Run `make init` to generate a default config or rename `config.example.yml` to `config.yml`
4. Configure `config.yml` with your Trac and GitLab settings
5. Run the commands in your terminal

**Trac Exporter:**

```bash
make export
```

**GitLab Importer:**

```bash
make migrate
```

## Restoring authors

The importer creates issues/notes with the admin token, because GitLab only lets
admins/owners set historical timestamps and IIDs. To restore the original
authors, emit the patch block and run it on the GitLab host:

```bash
# dry run: prints what would change
trac2gitlab patch

# after reviewing the dry run, emit an applying block
trac2gitlab patch --apply
```

Paste the emitted block into the GitLab host shell. It writes a temporary Ruby
file inside the container, runs `gitlab-rails runner`, and deletes the file.
Use `--container <name>` if the container is not called `gitlab`, or
`--omnibus` for a host install. The block updates `issues.author_id` and
`notes.author_id` only; unmapped authors are reported and skipped.

Check the `Makefile` for more commands and options.
