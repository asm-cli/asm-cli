package agent

import (
	"fmt"
	"path/filepath"
)

type Agent string

const (
	Claude Agent = "claude"
	Codex  Agent = "codex"
	Cursor Agent = "cursor"
	Gemini Agent = "gemini"
)

func Supported() []Agent {
	return []Agent{Claude, Codex, Cursor, Gemini}
}

func Parse(s string) (Agent, error) {
	switch Agent(s) {
	case Claude, Codex, Cursor, Gemini:
		return Agent(s), nil
	default:
		return "", fmt.Errorf("unsupported agent %q: must be one of claude, codex, cursor, gemini", s)
	}
}

func Home(a Agent, agentPaths map[string]string) (string, error) {
	p, ok := agentPaths[string(a)]
	if !ok || p == "" {
		return "", fmt.Errorf("no path configured for agent %q", a)
	}
	return p, nil
}

func SkillsDir(a Agent, agentPaths map[string]string) (string, error) {
	home, err := Home(a, agentPaths)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "skills"), nil
}

func MCPConfigPath(a Agent, agentPaths map[string]string) (string, error) {
	home, err := Home(a, agentPaths)
	if err != nil {
		return "", err
	}
	switch a {
	case Claude:
		return filepath.Join(home, "claude_desktop_config.json"), nil
	default:
		return filepath.Join(home, "mcp.json"), nil
	}
}
