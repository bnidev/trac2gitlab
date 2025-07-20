# Trac2GitLab

A tool to synchronize tickets, wikis and related data from a Trac instance (via XML-RPC) into a GitLab project.

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
