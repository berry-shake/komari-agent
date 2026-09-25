package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPrivateConfigExportPreservesArgumentBoundaries(t *testing.T) {
	previous := *flags
	previousOutput := writeConfig
	defer func() { *flags = previous; writeConfig = previousOutput; RootCmd.SetArgs(nil) }()
	t.Setenv("AGENT_TOKEN", "test-token-export")
	target := filepath.Join(t.TempDir(), "agent.json")
	endpoint := "https://example.com/panel?literal=a&second=b"
	mounts := "/data/my files;/opt/test"
	RootCmd.SetArgs([]string{"--endpoint", endpoint, "--include-mountpoint", mounts, "--disable-auto-update", "--write-config", target})
	if err := RootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got["token"] != "test-token-export" || got["endpoint"] != endpoint || got["include_mountpoints"] != mounts || got["disable_auto_update"] != true {
		t.Fatal("config values changed")
	}
}
