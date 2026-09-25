package update

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	flags "github.com/komari-monitor/komari-agent/cmd/flags"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestUpdaterNeverInheritsInsecurePanelTLS(t *testing.T) {
	old := flags.GlobalConfig.IgnoreUnsafeCert
	flags.GlobalConfig.IgnoreUnsafeCert = true
	oldDefault := http.DefaultTransport
	http.DefaultTransport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	defer func() { flags.GlobalConfig.IgnoreUnsafeCert = old; http.DefaultTransport = oldDefault }()
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	srv.Config.ErrorLog = log.New(io.Discard, "", 0)
	srv.StartTLS()
	defer srv.Close()
	client := newUpdateClient()
	defer client.CloseIdleConnections()
	response, err := client.Get(srv.URL)
	if response != nil {
		response.Body.Close()
	}
	if err == nil {
		t.Fatal("updater accepted an untrusted certificate")
	}
}

func TestUpdateRedirectAndSizeLimits(t *testing.T) {
	for _, raw := range []string{"http://github.com/file", "https://github.com.evil.invalid/file", "https://user@github.com/file", "https://github.com:444/file", "file:///tmp/file"} {
		if trustedUpdateURL(raw) {
			t.Errorf("trusted %q", raw)
		}
	}
	client := newUpdateClient()
	client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{"http://github.com/insecure"}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
	})
	if _, err := downloadUpdateAsset(client, "https://github.com/file", 10); err == nil {
		t.Fatal("accepted insecure redirect")
	}
	client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, ContentLength: -1, Body: io.NopCloser(strings.NewReader("too-large")), Header: make(http.Header), Request: r}, nil
	})
	if _, err := downloadUpdateAsset(client, "https://github.com/file", 3); err == nil {
		t.Fatal("accepted oversized chunked asset")
	}
}

func TestReleaseVerificationBeforeReplacingBinary(t *testing.T) {
	asset := []byte("new executable fixture")
	checksum := fmt.Sprintf("%x  komari-agent-linux-amd64", sha256.Sum256(asset))
	const binaryURL = "https://github.com/berry-shake/komari-agent/releases/download/1.2.6/komari-agent-linux-amd64"
	release := &platformRelease{AssetURL: binaryURL, ChecksumURL: binaryURL + ".sha256", AssetByteSize: len(asset)}
	for _, valid := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "agent")
		if err := os.WriteFile(path, []byte("old executable fixture"), 0700); err != nil {
			t.Fatal(err)
		}
		client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			data := asset
			if strings.HasSuffix(r.URL.Path, ".sha256") {
				data = []byte(checksum)
				if !valid {
					data = []byte("invalid checksum")
				}
			}
			return &http.Response{StatusCode: 200, ContentLength: int64(len(data)), Body: io.NopCloser(bytes.NewReader(data)), Header: make(http.Header), Request: r}, nil
		})}
		err := installRelease(client, release, path)
		got, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if valid {
			if err != nil || !bytes.Equal(got, asset) {
				t.Fatalf("valid update failed: %s %v", got, err)
			}
		} else {
			if err == nil || string(got) != "old executable fixture" {
				t.Fatalf("invalid checksum replaced binary: %s %v", got, err)
			}
		}
	}
}
