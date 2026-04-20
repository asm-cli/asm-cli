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
	agents := agent.Supported()
	if len(agents) != 4 {
		t.Errorf("expected 4 supported agents, got %d", len(agents))
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
	if p == "" {
		t.Error("expected non-empty MCP config path")
	}
}
