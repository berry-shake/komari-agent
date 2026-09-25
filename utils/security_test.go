package utils

import (
	"bytes"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCredentialFileReplacementIsPrivate(t *testing.T) {
	p := filepath.Join(t.TempDir(), "credentials.json")
	if err := os.WriteFile(p, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := WritePrivateFile(p, []byte("new")); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil || string(b) != "new" {
		t.Fatalf("%s %v", b, err)
	}
	st, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && st.Mode().Perm() != 0600 {
		t.Fatalf("permissions=%o", st.Mode().Perm())
	}
}

func TestLogRedactionIncludesUpdatedCredentials(t *testing.T) {
	secret := "fake token+with/slash"
	var out bytes.Buffer
	w := NewRedactingWriter(&out, func() []string { return []string{secret} })
	message := "network error ?token=" + url.QueryEscape(secret) + " raw=" + secret
	if n, err := w.Write([]byte(message)); err != nil || n != len(message) {
		t.Fatalf("write %d %v", n, err)
	}
	if strings.Contains(out.String(), secret) || strings.Contains(out.String(), url.QueryEscape(secret)) {
		t.Fatal("credential leaked")
	}
	secret = "new-discovered-token"
	w.Write([]byte(secret))
	if strings.Contains(out.String(), secret) {
		t.Fatal("updated credential leaked")
	}
}
