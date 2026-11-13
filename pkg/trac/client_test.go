package trac

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bnidev/trac2gitlab/internal/config"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestBasicAuthTransport(t *testing.T) {
	var auth string
	transport := &basicAuthTransport{
		username: "user",
		password: "pass",
		base: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			auth = req.Header.Get("Authorization")
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
		}),
	}

	req, err := http.NewRequest(http.MethodPost, "https://trac.example.com/xmlrpc", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transport.RoundTrip(req); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(auth, "Basic ") {
		t.Fatalf("Authorization = %q", auth)
	}
}

// versionServer answers system.getAPIVersion with the given integers.
func versionServer(t *testing.T, parts []int) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var vals strings.Builder
		for _, p := range parts {
			fmt.Fprintf(&vals, "<value><int>%d</int></value>", p)
		}
		w.Header().Set("Content-Type", "text/xml")
		_, _ = fmt.Fprintf(w, `<?xml version="1.0"?><methodResponse><params><param><value><array><data>%s</data></array></value></param></params></methodResponse>`, vals.String())
	}))
	t.Cleanup(srv.Close)
	var cfg config.Config
	cfg.Trac.BaseURL, cfg.Trac.RPCPath = srv.URL, "/xmlrpc"
	client, err := NewTracClient(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestValidatePluginVersion(t *testing.T) {
	cases := []struct {
		name    string
		version []int
		wantErr string // "" means compatible
	}{
		{"minimum", []int{1, 1, 9}, ""},
		{"newer", []int{1, 2, 0}, ""},
		{"too old", []int{1, 1, 8}, "incompatible"},
		{"two parts", []int{1, 2}, "unexpected version format"},
		{"empty", nil, "unexpected version format"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := versionServer(t, tc.version).ValidatePluginVersion()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("got %v, want compatible", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("got %v, want error containing %q", err, tc.wantErr)
			}
		})
	}
}
