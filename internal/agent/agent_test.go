package agent_test

import (
	"testing"

	"github.com/6xiaowu9/asm/internal/agent"
)

func TestParse_Valid(t *testing.T) {
	for _, name := range []string{"claude", "codex", "cursor", "gemini"} {
		a, err := agent.Parse(name)
		if err != nil {
			t.Errorf("Parse(%q): %v", name, err)
		}
		if string(a) != name {
			t.Errorf("Parse(%q) = %q", name, a)
		}
	}
}

func TestParse_Invalid(t *testing.T) {
	_, err := agent.Parse("vscode")
	if err == nil {
		t.Error("expected error for unsupported agent")
	}
}

func TestSupported(t *testing.T) {
	want := map[agent.Agent]bool{
		agent.Claude: true, agent.Codex: true,
		agent.Cursor: true, agent.Gemini: true,
	}
	for _, a := range agent.Supported() {
		if !want[a] {
			t.Errorf("unexpected agent in Supported(): %q", a)
		}
		delete(want, a)
	}
	for a := range want {
		t.Errorf("agent %q missing from Supported()", a)
	}
}

func TestHome(t *testing.T) {
	paths := map[string]string{"claude": "/home/user/.claude"}
	home, err := agent.Home(agent.Claude, paths)
	if err != nil {
		t.Fatal(err)
	}
	if home != "/home/user/.claude" {
		t.Errorf("Home = %q", home)
	}
}

func TestHome_Missing(t *testing.T) {
	_, err := agent.Home(agent.Claude, map[string]string{})
	if err == nil {
		t.Error("expected error for missing agent path")
	}
}

func TestSkillsDir(t *testing.T) {
	paths := map[string]string{"claude": "/home/user/.claude"}
	dir, err := agent.SkillsDir(agent.Claude, paths)
	if err != nil {
		t.Fatal(err)
	}
	if dir != "/home/user/.claude/skills" {
		t.Errorf("SkillsDir = %q", dir)
	}
}

func TestMCPConfigPath_Claude(t *testing.T) {
	paths := map[string]string{"claude": "/home/user/.claude"}
	p, err := agent.MCPConfigPath(agent.Claude, paths)
	if err != nil {
		t.Fatal(err)
	}
	want := "/home/user/.claude/claude_desktop_config.json"
	if p != want {
		t.Errorf("MCPConfigPath(claude) = %q, want %q", p, want)
	}
}

func TestMCPConfigPath_NonClaude(t *testing.T) {
	paths := map[string]string{"codex": "/home/user/.codex"}
	p, err := agent.MCPConfigPath(agent.Codex, paths)
	if err != nil {
		t.Fatal(err)
	}
	want := "/home/user/.codex/mcp.json"
	if p != want {
		t.Errorf("MCPConfigPath(codex) = %q, want %q", p, want)
	}
}
