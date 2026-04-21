// Package migrator imports unmanaged agent packages into the ASM store.
// It scans one or more agents for packages not tracked by the store, deduplicates
// by package ID across agents, then imports each unique package and links it to
// every agent that had it.
package migrator

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/6xiaowu9/asm/internal/installer"
	"github.com/6xiaowu9/asm/internal/linker"
	"github.com/6xiaowu9/asm/internal/store"
)

// Candidate is a deduplicated, unmanaged package found in one or more agent directories.
type Candidate struct {
	ID         string
	Kind       store.PackageKind
	FoundIn    []string          // agent names where this ID was found
	SourcePath string            // path to import from (first occurrence, symlinks resolved)
	LinkPaths  map[string]string // agent → explicit link path (empty = use computed default)
	ConfigData []byte            // non-nil for MCP: raw JSON config to write to store
}

// ImportedPackage records a successful import.
type ImportedPackage struct {
	ID      string
	Kind    store.PackageKind
	Agents  []string
	Skipped bool // already in store — only links were added
}

// Result is the full outcome of a migration run.
type Result struct {
	Candidates []Candidate
	Imported   []ImportedPackage
	DryRun     bool
}

// Migrator orchestrates the scan + import flow.
type Migrator struct {
	asmHome string
}

// New creates a Migrator. asmHome is used to detect whether a source path is
// already inside the ASM store tree.
func New(asmHome string) *Migrator {
	return &Migrator{asmHome: asmHome}
}

// Scan collects unmanaged packages across all specified agents and deduplicates
// by package ID. agentNames must be non-empty. The returned candidates are sorted
// by ID.
func (m *Migrator) Scan(
	s *store.Store,
	lnk *linker.Linker,
	agentNames []string,
) ([]Candidate, error) {
	// id → candidate (merged)
	byID := make(map[string]*Candidate)

	for _, agentName := range agentNames {
		var raw []linker.MigrationCandidate
		var err error
		switch s.Kind() {
		case store.PackageKindPlugin:
			raw, err = lnk.MigratePlugins(agentName)
		case store.PackageKindMCP:
			raw, err = lnk.MigrateMCPs(agentName)
		default:
			raw, err = lnk.Migrate(agentName)
		}
		if err != nil {
			return nil, fmt.Errorf("scan %s: %w", agentName, err)
		}
		for _, mc := range raw {
			if c, ok := byID[mc.ID]; ok {
				// Already seen from another agent — merge FoundIn and link paths.
				c.FoundIn = append(c.FoundIn, agentName)
				if mc.LinkPath != "" {
					if c.LinkPaths == nil {
						c.LinkPaths = make(map[string]string)
					}
					c.LinkPaths[agentName] = mc.LinkPath
				}
			} else {
				cand := &Candidate{
					ID:         mc.ID,
					Kind:       s.Kind(),
					FoundIn:    []string{agentName},
					ConfigData: mc.ConfigData,
				}
				if mc.ConfigData == nil {
					cand.SourcePath = resolveSource(mc.SourcePath)
				}
				if mc.LinkPath != "" {
					cand.LinkPaths = map[string]string{agentName: mc.LinkPath}
				}
				byID[mc.ID] = cand
			}
		}
	}

	out := make([]Candidate, 0, len(byID))
	for _, c := range byID {
		out = append(out, *c)
	}
	return out, nil
}

// Apply imports each candidate into the store and enables it for all agents
// that had it. Packages already present in the store are not re-imported; their
// new agents are still linked.
func (m *Migrator) Apply(
	candidates []Candidate,
	inst *installer.Installer,
	lnk *linker.Linker,
	dryRun bool,
) (Result, error) {
	result := Result{Candidates: candidates, DryRun: dryRun}

	for _, c := range candidates {
		if dryRun {
			result.Imported = append(result.Imported, ImportedPackage{
				ID: c.ID, Kind: c.Kind, Agents: c.FoundIn,
			})
			continue
		}

		alreadyInStore := false

		if c.ConfigData != nil {
			// MCP path: write config JSON directly to store.
			_, err := inst.InstallMCPConfig(c.ID, c.ConfigData)
			if err != nil {
				var aee *store.AlreadyExistsError
				if errors.As(err, &aee) {
					alreadyInStore = true
				} else {
					return result, fmt.Errorf("import %s: %w", c.ID, err)
				}
			}
		} else {
			// Skip import if source is already inside the ASM store tree.
			storeTree := filepath.Join(m.asmHome, "store")
			if strings.HasPrefix(c.SourcePath, storeTree) {
				alreadyInStore = true
			}
			if !alreadyInStore {
				_, err := inst.Install(c.SourcePath, installer.Options{ID: c.ID})
				if err != nil {
					var aee *store.AlreadyExistsError
					if errors.As(err, &aee) {
						alreadyInStore = true
					} else {
						return result, fmt.Errorf("import %s: %w", c.ID, err)
					}
				}
			}
		}

		// Enable for every agent that had this package.
		for _, agentName := range c.FoundIn {
			var enableErr error
			if c.ConfigData != nil {
				// MCP: already in agent's native config, just record ownership.
				enableErr = lnk.EnableNative(c.ID, agentName)
			} else if lp, ok := c.LinkPaths[agentName]; ok && lp != "" {
				enableErr = lnk.EnableAtPath(c.ID, agentName, lp)
			} else {
				enableErr = lnk.Enable(c.ID, agentName)
			}
			if enableErr != nil {
				return result, fmt.Errorf("enable %s for %s: %w", c.ID, agentName, enableErr)
			}
		}

		result.Imported = append(result.Imported, ImportedPackage{
			ID:      c.ID,
			Kind:    c.Kind,
			Agents:  c.FoundIn,
			Skipped: alreadyInStore,
		})
	}

	return result, nil
}

// resolveSource follows symlinks to get the real path to import from.
// Falls back to the original path on error.
func resolveSource(path string) string {
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	return real
}
