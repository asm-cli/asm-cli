package initializer

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/asm-cli/asm-cli/internal/agent"
	"github.com/asm-cli/asm-cli/internal/config"
)

// Options parameterises a Run call.
type Options struct {
	AsmHome string        // resolved ASM home path (e.g. ~/.asm)
	Agents  []agent.Agent // empty = auto-detect from home directory existence
	Force   bool          // overwrite existing per-agent ASM.md files
	DryRun  bool          // report what would happen without writing anything
	HomeDir string        // user home dir; overridable in tests
}

// AgentResult captures what happened for one agent during init.
type AgentResult struct {
	Agent      agent.Agent
	Skipped    bool // home directory does not exist
	ASMmdWrote bool // per-agent ASM.md was (or would be) written
	Injected   bool // @ASM.md line was (or would be) added to the agent config
}

// Result is the complete outcome of a Run call.
type Result struct {
	AsmHome     string
	ConfigWrote bool
	Agents      []AgentResult
	DryRun      bool
}

// Initializer performs the asm init sequence.
type Initializer struct{}

// New creates a new Initializer.
func New() *Initializer { return &Initializer{} }

// Run executes the full init sequence.
func (in *Initializer) Run(opts Options) (Result, error) {
	result := Result{AsmHome: opts.AsmHome, DryRun: opts.DryRun}

	// 1. Create directory tree.
	for _, sub := range []string{
		filepath.Join("store", "skills"),
		filepath.Join("store", "mcps"),
		filepath.Join("store", "plugins"),
		filepath.Join("cache", "git"),
	} {
		if !opts.DryRun {
			if err := os.MkdirAll(filepath.Join(opts.AsmHome, sub), 0o755); err != nil {
				return Result{}, fmt.Errorf("create %s: %w", sub, err)
			}
		}
	}

	// 2. Write config.toml if absent.
	cfgPath := filepath.Join(opts.AsmHome, "config.toml")
	if _, err := os.Stat(cfgPath); errors.Is(err, fs.ErrNotExist) {
		if !opts.DryRun {
			cfg := config.Default(opts.HomeDir)
			cfg.ASMHome = opts.AsmHome
			if err := config.Save(cfgPath, cfg); err != nil {
				return Result{}, fmt.Errorf("write config.toml: %w", err)
			}
		}
		result.ConfigWrote = true
	}

	// 3. Write canonical ~/.asm/ASM.md (always refreshed).
	if !opts.DryRun {
		asmMDPath := filepath.Join(opts.AsmHome, "ASM.md")
		if err := os.WriteFile(asmMDPath, []byte(asmMDContent(opts.AsmHome)), 0o644); err != nil {
			return Result{}, fmt.Errorf("write ASM.md: %w", err)
		}
	}

	// 4. Resolve target agents.
	agents := resolveAgents(opts)

	// 5. Process each agent.
	paths := defaultPaths(opts.HomeDir)
	for _, a := range agents {
		ar, err := processAgent(a, paths, opts)
		if err != nil {
			return Result{}, fmt.Errorf("agent %s: %w", a, err)
		}
		result.Agents = append(result.Agents, ar)
	}

	return result, nil
}

// resolveAgents returns agents to process. If opts.Agents is non-empty, it is
// used as-is. Otherwise all supported agents are returned; processAgent will
// mark absent ones as skipped.
func resolveAgents(opts Options) []agent.Agent {
	if len(opts.Agents) > 0 {
		return opts.Agents
	}
	return agent.Supported()
}

// defaultPaths derives agent home paths from homeDir without reading any config file,
// avoiding a bootstrap chicken-and-egg problem during init.
func defaultPaths(homeDir string) map[string]string {
	return map[string]string{
		"claude": filepath.Join(homeDir, ".claude"),
		"codex":  filepath.Join(homeDir, ".codex"),
		"cursor": filepath.Join(homeDir, ".cursor"),
		"gemini": filepath.Join(homeDir, ".gemini"),
	}
}

// processAgent handles one agent: injects the ASM.md reference into the agent's
// config markdown. For cursor, creates rules/ASM.md pointing at the canonical file.
// No per-agent copy of ASM.md is written; the canonical file lives at asmHome/ASM.md.
func processAgent(a agent.Agent, paths map[string]string, opts Options) (AgentResult, error) {
	ar := AgentResult{Agent: a}

	agentHome, ok := paths[string(a)]
	if !ok || agentHome == "" {
		ar.Skipped = true
		return ar, nil
	}

	if _, err := os.Stat(agentHome); errors.Is(err, fs.ErrNotExist) {
		ar.Skipped = true
		return ar, nil
	}

	// Cursor has no top-level config markdown; write a rules file instead.
	if a == agent.Cursor {
		dst := filepath.Join(agentHome, "rules", "ASM.md")
		_, statErr := os.Stat(dst)
		if errors.Is(statErr, fs.ErrNotExist) || opts.Force {
			if !opts.DryRun {
				if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
					return AgentResult{}, fmt.Errorf("create rules dir: %w", err)
				}
				if err := os.WriteFile(dst, []byte(asmMDContent(opts.AsmHome)), 0o644); err != nil {
					return AgentResult{}, fmt.Errorf("write rules/ASM.md: %w", err)
				}
			}
			ar.ASMmdWrote = true
		}
		return ar, nil
	}

	// For all other agents inject "@<asmHome>/ASM.md" into their config markdown.
	ref := "@" + filepath.Join(opts.AsmHome, "ASM.md")
	configPath := agentConfigFile(a, agentHome)
	injected, err := injectRef(configPath, ref, opts.DryRun)
	if err != nil {
		return AgentResult{}, fmt.Errorf("inject ref: %w", err)
	}
	ar.Injected = injected

	return ar, nil
}

// agentConfigFile returns the path of the config markdown for an agent.
func agentConfigFile(a agent.Agent, agentHome string) string {
	switch a {
	case agent.Claude:
		return filepath.Join(agentHome, "CLAUDE.md")
	case agent.Codex:
		return filepath.Join(agentHome, "AGENTS.md")
	case agent.Gemini:
		return filepath.Join(agentHome, "GEMINI.md")
	default:
		return ""
	}
}

// injectRef appends line to configPath if not already present.
// Returns true if the line was (or would be, in dry-run) added.
func injectRef(configPath, line string, dryRun bool) (bool, error) {
	data, err := os.ReadFile(configPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}

	// Check for existing reference (idempotent).
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == line {
			return false, nil
		}
	}

	if dryRun {
		return true, nil
	}

	// Ensure the file ends with a newline before appending.
	content := string(data)
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += line + "\n"

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return false, err
	}
	return true, os.WriteFile(configPath, []byte(content), 0o644)
}

// asmMDContent returns the ASM.md instruction file content.
func asmMDContent(asmHome string) string {
	_ = asmHome // reserved for future path interpolation
	return `# Agent Skills Manager (ASM)

ASM manages skill, MCP, and plugin packages for AI agent environments.
It installs packages into a central store and projects them into each agent's
home directory via symlinks (skills/plugins) or JSON manifests (MCP).

## IMPORTANT — always use ASM for package operations

**You MUST use the ` + "`asm`" + ` CLI for all install / enable / disable / update / remove
operations on skills, MCP servers, and plugins.**

Rules:
- ` + "`asm`" + ` is a **globally installed binary** — never use ` + "`npm`" + `, or any
  agent-specific install mechanism. Just run ` + "`asm <subcommand>`" + `.
- Before installing, check if the package is already in the store:
  ` + "`asm plugins list`" + ` / ` + "`asm skills list`" + `. If it is, use ` + "`enable`" + ` instead of ` + "`install`" + `.
- Never read the asm source code or search the current project to figure out
  how asm works — use ` + "`asm --help`" + ` or ` + "`asm <command> --help`" + ` instead.
- Never copy files manually — doing so bypasses the shared store and breaks
  cross-agent deduplication.

## Configuration

    ~/.asm/config.toml

## Slash command mapping

    /skills   →  asm skills  <subcommand>
    /mcp      →  asm mcp     <subcommand>
    /plugins  →  asm plugins <subcommand>

## Skills commands

    install <src> [--agents a,b] [--id id] [--ref ref] [--subdir dir]
    list
    status  [--agent a]
    enable  <id> [--agent a]
    disable <id> [--agent a]
    use     <id> [--agents a,b]
    update  <id> | --all
    remove  <id>
    sync
    doctor
    migrate [--agent a]

## MCP commands  (same subcommand set, operates on MCP packages)

    install / list / status / enable / disable / use / update / remove / sync / doctor / migrate

## Plugins commands  (same subcommand set, operates on agent plugins)

    install / list / status / enable / disable / use / update / remove / sync / doctor / migrate

## Typical workflow

` + "```" + `sh
asm init
asm skills  install https://github.com/owner/skill --agents claude
asm plugins install https://github.com/owner/plugin --agents claude,codex
asm skills  doctor
asm skills  update my-skill
` + "```" + `

## Notes

- asm init is idempotent: @ASM.md is never injected twice.
- asm skills sync recreates missing projections without reinstalling.
- Plugins are agent-native extensions (e.g. superpowers); ASM stores them once
  and symlinks each agent's plugin directory to the shared store copy.
`
}
