package config

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	ASMHome       string
	DefaultAgents []string
	LinkMode      string
	GitCacheDir   string
	AgentPaths    map[string]string
}

func Default(homeDir string) Config {
	asmHome := filepath.Join(homeDir, ".asm")
	return Config{
		ASMHome:       asmHome,
		DefaultAgents: []string{"claude"},
		LinkMode:      "auto",
		GitCacheDir:   filepath.Join(asmHome, "cache", "git"),
		AgentPaths: map[string]string{
			"claude": filepath.Join(homeDir, ".claude"),
			"codex":  filepath.Join(homeDir, ".codex"),
			"cursor": filepath.Join(homeDir, ".cursor"),
			"gemini": filepath.Join(homeDir, ".gemini"),
		},
	}
}

func Load(path string) (Config, error) {
	homeDir, _ := os.UserHomeDir()
	cfg := Default(homeDir)

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cfg, nil
		}
		return Config{}, err
	}

	section := ""
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.Trim(line, "[]")
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := trimQuotes(strings.TrimSpace(parts[1]))
		switch section {
		case "agent_paths":
			cfg.AgentPaths[key] = val
		default:
			switch key {
			case "asm_home":
				cfg.ASMHome = val
			case "link_mode":
				cfg.LinkMode = val
			case "git_cache_dir":
				cfg.GitCacheDir = val
			case "default_agents":
				cfg.DefaultAgents = parseArray(val)
			}
		}
	}
	return cfg, scanner.Err()
}

func Save(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "asm_home = %q\n", c.ASMHome)
	fmt.Fprintf(&buf, "link_mode = %q\n", c.LinkMode)
	fmt.Fprintf(&buf, "git_cache_dir = %q\n", c.GitCacheDir)
	fmt.Fprintf(&buf, "default_agents = [%s]\n\n", joinArray(c.DefaultAgents))
	buf.WriteString("[agent_paths]\n")
	for k, v := range c.AgentPaths {
		fmt.Fprintf(&buf, "%s = %q\n", k, v)
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func trimQuotes(s string) string { return strings.Trim(s, `"'`) }

func parseArray(s string) []string {
	s = strings.TrimSpace(strings.Trim(s, "[]"))
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, trimQuotes(strings.TrimSpace(p)))
	}
	return out
}

func joinArray(vs []string) string {
	parts := make([]string, len(vs))
	for i, v := range vs {
		parts[i] = fmt.Sprintf("%q", v)
	}
	return strings.Join(parts, ", ")
}
