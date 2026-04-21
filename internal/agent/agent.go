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

// MCPServerConfig holds the configuration for a single MCP server.
type MCPServerConfig struct {
	Type    string            `json:"type"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// MCPScanEntry describes an MCP server found in an agent's configuration.
type MCPScanEntry struct {
	ID         string
	ConfigJSON []byte // JSON-encoded MCPServerConfig
}

// MCPScanEntries returns MCP server entries found in the agent's native config.
func MCPScanEntries(a Agent, agentPaths map[string]string) ([]MCPScanEntry, error) {
	switch a {
	case Codex:
		return codexMCPEntries(agentPaths)
	case Claude:
		return claudeMCPEntries(agentPaths)
	default:
		return nil, nil
	}
}

func codexMCPEntries(agentPaths map[string]string) ([]MCPScanEntry, error) {
	home, ok := agentPaths[string(Codex)]
	if !ok || home == "" {
		return nil, nil
	}
	data, err := os.ReadFile(filepath.Join(home, "config.toml"))
	if err != nil {
		return nil, nil
	}

	// Parse [mcp_servers.<name>] and [mcp_servers.<name>.env] sections.
	servers := map[string]*MCPServerConfig{}
	var currentServer string
	var inEnv bool

	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := strings.Trim(line, "[]")
			if strings.HasPrefix(section, "mcp_servers.") {
				rest := strings.TrimPrefix(section, "mcp_servers.")
				if strings.Contains(rest, ".env") {
					// [mcp_servers.<name>.env]
					currentServer = strings.TrimSuffix(rest, ".env")
					inEnv = true
				} else {
					currentServer = rest
					inEnv = false
					if _, ok := servers[currentServer]; !ok {
						servers[currentServer] = &MCPServerConfig{}
					}
				}
			} else {
				currentServer = ""
				inEnv = false
			}
			continue
		}
		if currentServer == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		srv := servers[currentServer]
		if srv == nil {
			continue
		}
		if inEnv {
			if srv.Env == nil {
				srv.Env = map[string]string{}
			}
			srv.Env[key] = trimTOMLQuotes(val)
			continue
		}
		switch key {
		case "type":
			srv.Type = trimTOMLQuotes(val)
		case "command":
			srv.Command = trimTOMLQuotes(val)
		case "args":
			srv.Args = parseTOMLStringArray(val)
		}
	}

	var entries []MCPScanEntry
	for id, srv := range servers {
		data, err := json.Marshal(srv)
		if err != nil {
			continue
		}
		entries = append(entries, MCPScanEntry{ID: id, ConfigJSON: data})
	}
	return entries, nil
}

func claudeMCPEntries(agentPaths map[string]string) ([]MCPScanEntry, error) {
	home, ok := agentPaths[string(Claude)]
	if !ok || home == "" {
		return nil, nil
	}
	data, err := os.ReadFile(filepath.Join(home, "settings.json"))
	if err != nil {
		return nil, nil
	}

	var settings struct {
		MCPServers map[string]MCPServerConfig `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, nil
	}

	var entries []MCPScanEntry
	for id, srv := range settings.MCPServers {
		cfg := srv
		cfgJSON, err := json.Marshal(&cfg)
		if err != nil {
			continue
		}
		entries = append(entries, MCPScanEntry{ID: id, ConfigJSON: cfgJSON})
	}
	return entries, nil
}

func trimTOMLQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

func parseTOMLStringArray(s string) []string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "[]")
	if s == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(s, ",") {
		out = append(out, trimTOMLQuotes(strings.TrimSpace(part)))
	}
	return out
}

// MCPNativeConfigPath returns the file that ASM injects MCP config into.
func MCPNativeConfigPath(a Agent, agentPaths map[string]string) (string, error) {
	home, err := Home(a, agentPaths)
	if err != nil {
		return "", err
	}
	switch a {
	case Codex:
		return filepath.Join(home, "config.toml"), nil
	case Claude:
		return filepath.Join(home, "settings.json"), nil
	default:
		return "", fmt.Errorf("agent %q does not support MCP native config injection", a)
	}
}

// InjectMCP writes (or overwrites) the MCP server entry in the agent's native config.
func InjectMCP(a Agent, agentPaths map[string]string, id string, cfg MCPServerConfig) error {
	switch a {
	case Codex:
		home, err := Home(a, agentPaths)
		if err != nil {
			return err
		}
		return injectCodexMCP(home, id, cfg)
	case Claude:
		home, err := Home(a, agentPaths)
		if err != nil {
			return err
		}
		return injectClaudeMCP(home, id, cfg)
	default:
		return fmt.Errorf("agent %q does not support MCP injection", a)
	}
}

// RemoveMCP removes the MCP server entry from the agent's native config.
func RemoveMCP(a Agent, agentPaths map[string]string, id string) error {
	switch a {
	case Codex:
		home, err := Home(a, agentPaths)
		if err != nil {
			return err
		}
		return removeCodexMCP(home, id)
	case Claude:
		home, err := Home(a, agentPaths)
		if err != nil {
			return err
		}
		return removeClaudeMCP(home, id)
	default:
		return fmt.Errorf("agent %q does not support MCP removal", a)
	}
}

// injectCodexMCP upserts [mcp_servers.<id>] in ~/.codex/config.toml.
func injectCodexMCP(home, id string, cfg MCPServerConfig) error {
	cfgPath := filepath.Join(home, "config.toml")
	data, _ := os.ReadFile(cfgPath)

	cleaned := removeCodexMCPSections(string(data), id)

	var buf strings.Builder
	buf.WriteString(strings.TrimRight(cleaned, "\n"))
	buf.WriteString("\n\n")
	fmt.Fprintf(&buf, "[mcp_servers.%s]\n", id)
	fmt.Fprintf(&buf, "type = %q\n", cfg.Type)
	fmt.Fprintf(&buf, "command = %q\n", cfg.Command)
	if len(cfg.Args) > 0 {
		parts := make([]string, len(cfg.Args))
		for i, a := range cfg.Args {
			parts[i] = fmt.Sprintf("%q", a)
		}
		fmt.Fprintf(&buf, "args = [%s]\n", strings.Join(parts, ", "))
	}
	if len(cfg.Env) > 0 {
		fmt.Fprintf(&buf, "\n[mcp_servers.%s.env]\n", id)
		for k, v := range cfg.Env {
			fmt.Fprintf(&buf, "%s = %q\n", k, v)
		}
	}
	buf.WriteString("\n")

	return os.WriteFile(cfgPath, []byte(buf.String()), 0o644)
}

// removeCodexMCP removes [mcp_servers.<id>] sections from ~/.codex/config.toml.
func removeCodexMCP(home, id string) error {
	cfgPath := filepath.Join(home, "config.toml")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	cleaned := removeCodexMCPSections(string(data), id)
	return os.WriteFile(cfgPath, []byte(cleaned), 0o644)
}

// removeCodexMCPSections strips [mcp_servers.<id>] and [mcp_servers.<id>.env]
// sections from a config.toml string.
func removeCodexMCPSections(content, id string) string {
	prefix1 := "[mcp_servers." + id + "]"
	prefix2 := "[mcp_servers." + id + "."
	var out []string
	skip := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			skip = trimmed == prefix1 || strings.HasPrefix(trimmed, prefix2)
		}
		if !skip {
			out = append(out, line)
		}
	}
	result := strings.Join(out, "\n")
	// Collapse multiple blank lines into one.
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}
	return result
}

// injectClaudeMCP upserts the MCP server under mcpServers in ~/.claude/settings.json.
func injectClaudeMCP(home, id string, cfg MCPServerConfig) error {
	cfgPath := filepath.Join(home, "settings.json")
	data, _ := os.ReadFile(cfgPath)

	var settings map[string]json.RawMessage
	if len(data) > 0 {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("parse settings.json: %w", err)
		}
	} else {
		settings = make(map[string]json.RawMessage)
	}

	var mcpServers map[string]json.RawMessage
	if raw, ok := settings["mcpServers"]; ok {
		_ = json.Unmarshal(raw, &mcpServers)
	}
	if mcpServers == nil {
		mcpServers = make(map[string]json.RawMessage)
	}

	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	mcpServers[id] = cfgJSON

	raw, err := json.Marshal(mcpServers)
	if err != nil {
		return err
	}
	settings["mcpServers"] = raw

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfgPath, append(out, '\n'), 0o644)
}

// removeClaudeMCP removes the MCP server from mcpServers in ~/.claude/settings.json.
func removeClaudeMCP(home, id string) error {
	cfgPath := filepath.Join(home, "settings.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var settings map[string]json.RawMessage
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("parse settings.json: %w", err)
	}

	if raw, ok := settings["mcpServers"]; ok {
		var mcpServers map[string]json.RawMessage
		if err := json.Unmarshal(raw, &mcpServers); err == nil {
			delete(mcpServers, id)
			updated, _ := json.Marshal(mcpServers)
			settings["mcpServers"] = updated
		}
	}

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfgPath, append(out, '\n'), 0o644)
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
