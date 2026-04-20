package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/6xiaowu9/asm/internal/config"
)

func TestDefault(t *testing.T) {
	cfg := config.Default("/home/user")
	if cfg.ASMHome != "/home/user/.asm" {
		t.Errorf("ASMHome = %q, want /home/user/.asm", cfg.ASMHome)
	}
	if cfg.AgentPaths["claude"] != "/home/user/.claude" {
		t.Errorf("claude path = %q", cfg.AgentPaths["claude"])
	}
	if cfg.LinkMode != "auto" {
		t.Errorf("LinkMode = %q, want auto", cfg.LinkMode)
	}
}

func TestLoad_FileAbsent_ReturnsDefault(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Load(filepath.Join(dir, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LinkMode == "" {
		t.Error("expected default LinkMode")
	}
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	want := config.Default(dir)
	want.LinkMode = "symlink"
	want.DefaultAgents = []string{"claude", "codex"}
	want.AgentPaths["claude"] = "/custom/claude"

	if err := config.Save(path, want); err != nil {
		t.Fatal("Save:", err)
	}

	got, err := config.Load(path)
	if err != nil {
		t.Fatal("Load:", err)
	}

	if got.LinkMode != want.LinkMode {
		t.Errorf("LinkMode = %q, want %q", got.LinkMode, want.LinkMode)
	}
	if len(got.DefaultAgents) != 2 {
		t.Errorf("DefaultAgents = %v", got.DefaultAgents)
	}
	if got.AgentPaths["claude"] != "/custom/claude" {
		t.Errorf("AgentPaths[claude] = %q", got.AgentPaths["claude"])
	}
}

func TestLoad_CorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	_ = os.WriteFile(path, []byte("not = [valid toml"), 0o644)
	cfg, err := config.Load(path)
	// Must not panic; either an error or a config with defaults is acceptable.
	if err == nil && cfg.LinkMode == "" {
		t.Error("corrupt file: expected either an error or a config with defaults")
	}
}
