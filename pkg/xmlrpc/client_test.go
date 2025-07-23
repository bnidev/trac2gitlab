package xmlrpc

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient_BasicCallBehavior(t *testing.T) {
	// Create a fake XML-RPC server that returns a known response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		if !bytes.Contains(body, []byte("sample.method")) {
			t.Errorf("expected method 'sample.method' in request body, got: %s", body)
		}

		// Send a dummy XML-RPC response
		response := `<?xml version="1.0"?>
<methodResponse>
  <params>
    <param>
      <value><string>Hello, world!</string></value>
    </param>
  </params>
</methodResponse>`
		w.Header().Set("Content-Type", "text/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, nil)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	var result string
	err = client.Call("sample.method", nil, &result)
	if err != nil {
		t.Fatalf("client.Call returned error: %v", err)
	}

	expected := "Hello, world!"
	if result != expected {
		t.Errorf("expected result %q, got %q", expected, result)
	}
}

func TestNewClient_InvalidURL(t *testing.T) {
	_, err := NewClient(":", nil)
	if err == nil {
		t.Fatalf("expected error for invalid URL, got nil")
	}
}
