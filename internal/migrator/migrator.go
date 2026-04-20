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
	FoundIn    []string // agent names where this ID was found
	SourcePath string   // path to import from (first occurrence, symlinks resolved)
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
		raw, err := lnk.Migrate(agentName)
		if err != nil {
			return nil, fmt.Errorf("scan %s: %w", agentName, err)
		}
		for _, mc := range raw {
			if c, ok := byID[mc.ID]; ok {
				// Already seen from another agent — merge FoundIn only.
				c.FoundIn = append(c.FoundIn, agentName)
			} else {
				// Resolve symlinks so we import real content, not a dangling pointer.
				src := resolveSource(mc.SourcePath)
				byID[mc.ID] = &Candidate{
					ID:         mc.ID,
					Kind:       s.Kind(),
					FoundIn:    []string{agentName},
					SourcePath: src,
				}
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

		// Enable for every agent that had this package.
		for _, agentName := range c.FoundIn {
			if err := lnk.Enable(c.ID, agentName); err != nil {
				return result, fmt.Errorf("enable %s for %s: %w", c.ID, agentName, err)
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
