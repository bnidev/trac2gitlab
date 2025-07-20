package xmlrpc

import (
	"net/http"

	"github.com/kolo/xmlrpc"
)

type Client struct {
	*xmlrpc.Client
}

// NewClient creates a new XML-RPC client for a given endpoint URL.
func NewClient(url string, transport http.RoundTripper) (*Client, error) {
	if transport == nil {
		transport = http.DefaultTransport
	}

	client, err := xmlrpc.NewClient(url, transport)
	if err != nil {
		return nil, err
	}

	return &Client{client}, nil
}
