package utils

import (
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const MaxMessageBytes int64 = 1 << 20

func ReadBounded(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("response exceeds size limit")
	}
	return data, nil
}

func WritePrivateFile(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".komari-credentials-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = ProtectCredentialFile(f.Name()); err != nil {
		f.Close()
		return err
	}
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

type redactingWriter struct {
	target  io.Writer
	secrets func() []string
}

func NewRedactingWriter(target io.Writer, secrets func() []string) io.Writer {
	return &redactingWriter{target: target, secrets: secrets}
}

func (w *redactingWriter) Write(p []byte) (int, error) {
	message := string(p)
	for _, secret := range w.secrets() {
		if secret == "" {
			continue
		}
		for _, value := range []string{url.QueryEscape(secret), url.PathEscape(secret), secret} {
			message = strings.ReplaceAll(message, value, "[REDACTED]")
		}
	}
	_, err := io.WriteString(w.target, message)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}
