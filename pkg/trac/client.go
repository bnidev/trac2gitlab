package trac

import (
	"fmt"
	"log/slog"
	"slices"
	"trac2gitlab/pkg/xmlrpc"
)

type Client struct {
	rpc *xmlrpc.Client
}

// NewTracClient creates a new Trac XML-RPC client.
func NewTracClient(baseURL, rpcPath string) (*Client, error) {
	url := fmt.Sprintf("%s%s", baseURL, rpcPath)

	slog.Debug("Creating Trac XML-RPC client", "url", url)

	rpcClient, err := xmlrpc.NewClient(url, nil)
	if err != nil {
		return nil, err
	}
	return &Client{rpc: rpcClient}, nil
}

// CheckPluginVersion checks the Trac XML-RPC plugin version.
func (c *Client) CheckPluginVersion() ([]int64, error) {
	var version []int64
	err := c.rpc.Call("system.getAPIVersion", nil, &version)
	return version, err
}

// GetAvailableMethods retrieves the list of available XML-RPC methods from the Trac server.
func (c *Client) GetAvailableMethods() ([]string, error) {
	var methods []string
	err := c.rpc.Call("system.listMethods", nil, &methods)
	if err != nil {
		return nil, err
	}

	return methods, nil
}

// GetAttachmentList retrieves the list of attachments for a given ticket ID.
func (c *Client) ValidateExpectedMethods() error {
	methods, err := c.GetAvailableMethods()
	if err != nil {
		return fmt.Errorf("failed to get available methods: %w", err)
	}

	expected := []string{"ticket.get", "ticket.query", "wiki.getPage"}
	for _, want := range expected {
		if !slices.Contains(methods, want) {
			return fmt.Errorf("missing expected method: %s", want)
		}
	}

	slog.Debug("Trac XML-RPC plugin method validation successful. All expected methods are available.")
	return nil
}

// ValidatePluginVersion checks if the Trac XML-RPC plugin version is compatible.
func (c *Client) ValidatePluginVersion() error {
	version, err := c.CheckPluginVersion()
	if err != nil {
		return fmt.Errorf("failed to check plugin version: %w", err)
	}

	if len(version) < 2 {
		return fmt.Errorf("unexpected version format: %v", version)
	}

	epoch, major, minor := version[0], version[1], version[2]

	// Required minimum version
	const requiredEpoch = 1
	const requiredMajor = 1
	const requiredMinor = 9

	if epoch < requiredEpoch ||
		(epoch == requiredEpoch && major < requiredMajor) ||
		(epoch == requiredEpoch && major == requiredMajor && minor < requiredMinor) {
		return fmt.Errorf(
			"incompatible Trac XML-RPC plugin version: %d.%d.%d (minimum required: %d.%d.%d)",
			epoch, major, minor, requiredEpoch, requiredMajor, requiredMinor,
		)
	}

	formattedVersion := fmt.Sprintf("%d.%d.%d", epoch, major, minor)
	slog.Debug("Trac XML-RPC plugin version check successful", "version", formattedVersion)
	return nil
}
