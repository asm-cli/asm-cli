// Package ejector restores ASM-managed packages back to agent directories,
// reversing the effect of migration. After ejection every agent has real
// directory copies (skills/plugins) and its own native config entries (MCPs)
// that no longer reference the ASM store.
package ejector

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/asm-cli/asm-cli/internal/agent"
	"github.com/asm-cli/asm-cli/internal/config"
	"github.com/asm-cli/asm-cli/internal/store"
)

// EjectedItem describes one restored package.
type EjectedItem struct {
	Kind      store.PackageKind
	PackageID string
	Agent     string
	Detail    string // human-readable description of what was done
}

// Result holds the full outcome of an eject run.
type Result struct {
	Items  []EjectedItem
	DryRun bool
}

// Eject iterates all tracked links across skills, MCP, and plugins and restores
// each package back into the relevant agent directory.
func Eject(asmHome string, cfg config.Config, dryRun bool) (Result, error) {
	var result Result
	result.DryRun = dryRun

	storeTree := filepath.Join(asmHome, "store")

	for _, kind := range []store.PackageKind{
		store.PackageKindSkill,
		store.PackageKindMCP,
		store.PackageKindPlugin,
	} {
		s := store.New(asmHome, kind)
		links, err := s.ListLinks()
		if err != nil {
			return result, fmt.Errorf("list %s links: %w", kind, err)
		}

		for _, lr := range links {
			var item EjectedItem
			var ejectErr error

			switch kind {
			case store.PackageKindSkill, store.PackageKindPlugin:
				item, ejectErr = ejectSymlink(lr, dryRun)
			case store.PackageKindMCP:
				item, ejectErr = ejectMCP(lr, cfg, storeTree, dryRun)
			}

			if ejectErr != nil {
				return result, fmt.Errorf("eject %s %s (%s): %w", kind, lr.PackageID, lr.Agent, ejectErr)
			}
			result.Items = append(result.Items, item)
		}
	}

	return result, nil
}

// ejectSymlink replaces the symlink at lr.LinkPath with a real copy of the
// store directory, so the agent no longer needs the ASM store at runtime.
func ejectSymlink(lr store.LinkRecord, dryRun bool) (EjectedItem, error) {
	item := EjectedItem{
		Kind:      lr.Kind,
		PackageID: lr.PackageID,
		Agent:     lr.Agent,
		Detail:    fmt.Sprintf("copy %s → %s", lr.Target, lr.LinkPath),
	}
	if dryRun {
		return item, nil
	}

	// Remove symlink (or existing directory).
	if err := os.RemoveAll(lr.LinkPath); err != nil && !isNotExist(err) {
		return item, fmt.Errorf("remove link: %w", err)
	}

	if err := copyDir(lr.Target, lr.LinkPath); err != nil {
		return item, fmt.Errorf("copy dir: %w", err)
	}
	return item, nil
}

// ejectMCP handles MCP packages. If the stored config has a command that points
// inside the ASM store (a local binary), the binary is copied to
// <agentHome>/bin/<binaryname> and the agent's native config is updated.
// For npx/http/sse servers nothing file-level needs to change.
func ejectMCP(lr store.LinkRecord, cfg config.Config, storeTree string, dryRun bool) (EjectedItem, error) {
	item := EjectedItem{
		Kind:      lr.Kind,
		PackageID: lr.PackageID,
		Agent:     lr.Agent,
	}

	// Read the stored config.json.
	cfgFile := filepath.Join(lr.Target, "config.json")
	data, err := os.ReadFile(cfgFile)
	if err != nil {
		item.Detail = "no local binary (skipped)"
		return item, nil
	}

	var mcpCfg agent.MCPServerConfig
	if err := json.Unmarshal(data, &mcpCfg); err != nil {
		return item, fmt.Errorf("parse config.json: %w", err)
	}

	// Only act when the command is a binary inside the ASM store tree.
	if mcpCfg.Command == "" || !strings.HasPrefix(mcpCfg.Command, storeTree) {
		item.Detail = "no local binary (skipped)"
		return item, nil
	}

	agentHome, ok := cfg.AgentPaths[lr.Agent]
	if !ok {
		return item, fmt.Errorf("no path configured for agent %q", lr.Agent)
	}

	binName := filepath.Base(mcpCfg.Command)
	destBin := filepath.Join(agentHome, "bin", binName)
	item.Detail = fmt.Sprintf("restore binary %s → %s", mcpCfg.Command, destBin)

	if dryRun {
		return item, nil
	}

	// Copy binary back to agent bin dir.
	if err := os.MkdirAll(filepath.Dir(destBin), 0o755); err != nil {
		return item, fmt.Errorf("mkdir bin: %w", err)
	}
	if err := copyFile(mcpCfg.Command, destBin); err != nil {
		return item, fmt.Errorf("copy binary: %w", err)
	}
	if info, err := os.Stat(mcpCfg.Command); err == nil {
		_ = os.Chmod(destBin, info.Mode())
	}

	// Update the command path in the agent's native config.
	a, err := agent.Parse(lr.Agent)
	if err != nil {
		return item, err
	}
	mcpCfg.Command = destBin
	if err := agent.InjectMCP(a, cfg.AgentPaths, lr.PackageID, mcpCfg); err != nil {
		return item, fmt.Errorf("update agent config: %w", err)
	}

	return item, nil
}

func isNotExist(err error) bool { return err != nil && os.IsNotExist(err) }

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
