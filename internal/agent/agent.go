package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// PluginScanEntry describes a plugin found in an agent's environment.
type PluginScanEntry struct {
	// ID is the canonical plugin name (e.g. "superpowers").
	ID string
	// SourcePath is the directory whose content should be copied to the store.
	SourcePath string
	// LinkPath is where the symlink back to the store should be created,
	// replacing the original directory.
	LinkPath string
}

// PluginScanEntries returns the unmanaged plugin directories for the given agent.
// Returns nil without error when the agent has no plugin directory.
func PluginScanEntries(a Agent, agentPaths map[string]string) ([]PluginScanEntry, error) {
	switch a {
	case Claude:
		return claudePluginEntries(agentPaths)
	case Codex:
		return codexPluginEntries(agentPaths)
	default:
		return nil, nil
	}
}

func claudePluginEntries(agentPaths map[string]string) ([]PluginScanEntry, error) {
	home, ok := agentPaths[string(Claude)]
	if !ok || home == "" {
		return nil, nil
	}
	manifestPath := filepath.Join(home, "plugins", "installed_plugins.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, nil // plugins not installed
	}

	var manifest struct {
		Plugins map[string][]struct {
			InstallPath string `json:"installPath"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse installed_plugins.json: %w", err)
	}

	var entries []PluginScanEntry
	for key, versions := range manifest.Plugins {
		if len(versions) == 0 || versions[0].InstallPath == "" {
			continue
		}
		// "superpowers@claude-plugins-official" → "superpowers"
		id := strings.SplitN(key, "@", 2)[0]
		installPath := versions[0].InstallPath
		entries = append(entries, PluginScanEntry{
			ID:         id,
			SourcePath: installPath,
			LinkPath:   installPath,
		})
	}
	return entries, nil
}

func codexPluginEntries(agentPaths map[string]string) ([]PluginScanEntry, error) {
	home, ok := agentPaths[string(Codex)]
	if !ok || home == "" {
		return nil, nil
	}
	dirEntries, err := os.ReadDir(home)
	if err != nil {
		return nil, nil
	}

	var entries []PluginScanEntry
	for _, e := range dirEntries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		// A directory is treated as a plugin when it contains a ".codex/" subdir,
		// which is the per-agent config marker written by the superpowers package.
		marker := filepath.Join(home, e.Name(), ".codex")
		if _, err := os.Stat(marker); err != nil {
			continue
		}
		pluginDir := filepath.Join(home, e.Name())
		entries = append(entries, PluginScanEntry{
			ID:         e.Name(),
			SourcePath: pluginDir,
			LinkPath:   pluginDir,
		})
	}
	return entries, nil
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
