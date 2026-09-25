package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReportsSendTokenInHeader(t *testing.T) {
	oldEndpoint, oldToken := flags.Endpoint, flags.Token
	defer func() { flags.Endpoint = oldEndpoint; flags.Token = oldToken }()
	flags.Token = "fake-node-secret"
	seen := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = true
		if r.Header.Get("X-Client-Token") != flags.Token {
			t.Error("missing token header")
		}
		if r.URL.Query().Get("token") != "" {
			t.Error("token leaked in URL")
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()
	flags.Endpoint = srv.URL
	if err := postV2RPC(map[string]string{"test": "fixture"}); err != nil {
		t.Fatal(err)
	}
	if !seen {
		t.Fatal("no report received")
	}
	if endpoint := buildWebSocketEndpoint(2); strings.Contains(endpoint, flags.Token) || strings.Contains(endpoint, "token=") {
		t.Fatal("websocket token leaked in URL")
	}
	if newWSHeaders().Get("X-Client-Token") != flags.Token {
		t.Fatal("missing websocket credential")
	}
}
