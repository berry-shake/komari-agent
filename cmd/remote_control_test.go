package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func exportRemoteControlConfig(t *testing.T, args ...string) map[string]interface{} {
	t.Helper()
	previous := *flags
	previousOutput := writeConfig
	defer func() { *flags = previous; writeConfig = previousOutput; RootCmd.SetArgs(nil) }()
	target := filepath.Join(t.TempDir(), "agent.json")
	RootCmd.SetArgs(append(append([]string{"-e", "https://example.com", "-t", "test-token"}, args...), "--write-config", target))
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
	return got
}

func TestRemoteControlDisabledByDefault(t *testing.T) {
	got := exportRemoteControlConfig(t)
	if got["enable_web_ssh"] != false || got["disable_web_ssh"] != false {
		t.Fatalf("unexpected defaults: enable=%v disable=%v", got["enable_web_ssh"], got["disable_web_ssh"])
	}
}

func TestRemoteControlEnableFlagExported(t *testing.T) {
	got := exportRemoteControlConfig(t, "--enable-web-ssh")
	if got["enable_web_ssh"] != true {
		t.Fatalf("enable_web_ssh not exported: %v", got["enable_web_ssh"])
	}
}

func TestRemoteControlEnabledResolution(t *testing.T) {
	cases := []struct {
		name            string
		enable, disable bool
		want            bool
	}{
		{"default", false, false, false},
		{"enabled", true, false, true},
		{"legacy disable only", false, true, false},
		{"disable overrides enable", true, true, false},
	}
	for _, c := range cases {
		cfg := *flags
		cfg.EnableWebSsh, cfg.DisableWebSsh = c.enable, c.disable
		if got := cfg.RemoteControlEnabled(); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestRemoteControlEnabledFromEnv(t *testing.T) {
	previous := *flags
	defer func() { *flags = previous }()
	flags.EnableWebSsh, flags.DisableWebSsh = false, false
	t.Setenv("AGENT_ENABLE_WEB_SSH", "true")
	loadFromEnv()
	if !flags.RemoteControlEnabled() {
		t.Fatal("AGENT_ENABLE_WEB_SSH=true should enable remote control")
	}
}

func TestRemoteControlEnabledFromConfigFile(t *testing.T) {
	var cfg = *flags
	cfg.EnableWebSsh, cfg.DisableWebSsh = false, false
	if err := json.Unmarshal([]byte(`{"enable_web_ssh": true}`), &cfg); err != nil {
		t.Fatal(err)
	}
	if !cfg.RemoteControlEnabled() {
		t.Fatal("enable_web_ssh in config file should enable remote control")
	}
}
