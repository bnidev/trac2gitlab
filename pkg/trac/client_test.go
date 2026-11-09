package trac

import (
	"io"
	"net/http"
	"strings"
	"testing"
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
