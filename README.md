# Trac2GitLab

A tool to synchronize tickets, wikis and related data from a Trac instance (via XML-RPC) into a GitLab project.

## Current Features

### Exporter

- Export tickets from Trac as JSON files (including history)
- Export content of wiki pages from Trac as markdown files (including history)
- Export metadata of wiki pages from Trac as JSON files (including history)
- Export milestones as JSON files
- Concurrent export operations for faster migration (speed might be limited by Trac XML-RPC)
- Download attachments
- Configurable via YAML

### Importer

- Import milestones into GitLab projects (updates if already exist and content differs)

## Planned Features

### Exporter

- Export comments (not supported by the Trac XML-RPC plugin)

### Importer

- Import issues into GitLab projects
- Preserve ticket history in GitLab issues (not supported by the GitLab API)
- Map Trac fields to GitLab labels
- Import wiki pages into GitLab projects
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

## Running the tool

1. Clone the repository
2. Configure `config.example.yml` to your environment and rename it to `config.yml`
3. Run the exporter via:

```bash
go run ./cmd/trac2gitlab export
```
