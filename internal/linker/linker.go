package linker

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/6xiaowu9/asm/internal/agent"
	"github.com/6xiaowu9/asm/internal/store"
)

// StatusReport holds the projection status for an agent.
type StatusReport struct {
	Agent             string
	Kind              store.PackageKind
	InstalledPackages []store.PackageRecord
	EnabledPackages   []store.PackageRecord
	DisabledPackages  []store.PackageRecord
}

// Issue represents a detected problem with a projection.
type Issue struct {
	PackageID string
	Agent     string
	Problem   string
}

// MigrationCandidate is a package-like entry found in the agent directory
// that is not yet tracked by the store.
type MigrationCandidate struct {
	ID         string
	SourcePath string
}

// Linker manages projections (symlinks / JSON manifests) between the ASM
// store and agent-specific directories.
type Linker struct {
	store      *store.Store
	agentPaths map[string]string
}

// New creates a new Linker.
func New(s *store.Store, agentPaths map[string]string) *Linker {
	return &Linker{store: s, agentPaths: agentPaths}
}

// linkPath computes the filesystem path where the projection for rec should
// live inside the agent's home directory.
func (l *Linker) linkPath(rec store.PackageRecord, a agent.Agent) (string, error) {
	switch rec.Kind {
	case store.PackageKindSkill:
		skillsDir, err := agent.SkillsDir(a, l.agentPaths)
		if err != nil {
			return "", err
		}
		return filepath.Join(skillsDir, rec.ID), nil
	default: // MCP
		home, err := agent.Home(a, l.agentPaths)
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "mcp", rec.ID+".json"), nil
	}
}

// writeMCPManifest writes a small JSON manifest for an MCP package.
func writeMCPManifest(path string, rec store.PackageRecord) error {
	type manifest struct {
		ID         string `json:"id"`
		StorePath  string `json:"store_path"`
		ManagedBy  string `json:"managed_by"`
	}
	data, err := json.MarshalIndent(manifest{ID: rec.ID, StorePath: rec.StorePath, ManagedBy: "asm"}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Link creates (or recreates) the projection for packageID in the given agent.
func (l *Linker) Link(packageID, agentName string) error {
	rec, ok := l.store.GetPackage(packageID)
	if !ok {
		return &store.NotFoundError{ID: packageID}
	}

	a, err := agent.Parse(agentName)
	if err != nil {
		return err
	}

	lp, err := l.linkPath(rec, a)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(lp), 0o755); err != nil {
		return err
	}

	// Remove any existing entry (ignore error).
	_ = os.Remove(lp)

	var mode string
	switch rec.Kind {
	case store.PackageKindSkill:
		if err := os.Symlink(rec.StorePath, lp); err != nil {
			return err
		}
		mode = "symlink"
	default:
		if err := writeMCPManifest(lp, rec); err != nil {
			return err
		}
		mode = "generate"
	}

	lr := store.LinkRecord{
		PackageID: packageID,
		Agent:     agentName,
		Kind:      rec.Kind,
		Mode:      mode,
		LinkPath:  lp,
		Target:    rec.StorePath,
		UpdatedAt: time.Now(),
	}
	return l.store.SaveLink(lr)
}

// Unlink removes the projection for packageID from the given agent.
func (l *Linker) Unlink(packageID, agentName string) error {
	links, err := l.store.GetLinks(packageID)
	if err != nil {
		return err
	}

	var lp string
	for _, lr := range links {
		if lr.Agent == agentName {
			lp = lr.LinkPath
			break
		}
	}

	if lp != "" {
		if err := os.Remove(lp); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}

	return l.store.DeleteLink(packageID, agentName)
}

// addEnabledAgent appends agentName to rec.EnabledAgents if not already present.
func (l *Linker) addEnabledAgent(packageID, agentName string) error {
	rec, ok := l.store.GetPackage(packageID)
	if !ok {
		return &store.NotFoundError{ID: packageID}
	}
	for _, a := range rec.EnabledAgents {
		if a == agentName {
			return nil
		}
	}
	rec.EnabledAgents = append(rec.EnabledAgents, agentName)
	return l.store.SavePackage(rec)
}

// removeEnabledAgent filters agentName out of rec.EnabledAgents.
func (l *Linker) removeEnabledAgent(packageID, agentName string) error {
	rec, ok := l.store.GetPackage(packageID)
	if !ok {
		return &store.NotFoundError{ID: packageID}
	}
	filtered := rec.EnabledAgents[:0]
	for _, a := range rec.EnabledAgents {
		if a != agentName {
			filtered = append(filtered, a)
		}
	}
	rec.EnabledAgents = filtered
	return l.store.SavePackage(rec)
}

// Enable creates the projection and marks the agent as enabled.
func (l *Linker) Enable(packageID, agentName string) error {
	if err := l.Link(packageID, agentName); err != nil {
		return err
	}
	return l.addEnabledAgent(packageID, agentName)
}

// Disable removes the projection and marks the agent as disabled.
func (l *Linker) Disable(packageID, agentName string) error {
	if err := l.Unlink(packageID, agentName); err != nil {
		return err
	}
	return l.removeEnabledAgent(packageID, agentName)
}

// Use enables packageID for each agent in the agents slice.
func (l *Linker) Use(packageID string, agents []string) error {
	for _, a := range agents {
		if err := l.Enable(packageID, a); err != nil {
			return err
		}
	}
	return nil
}

// Migrate scans the agent's package directories and returns entries whose IDs
// are not already in the store.
func (l *Linker) Migrate(agentName string) ([]MigrationCandidate, error) {
	a, err := agent.Parse(agentName)
	if err != nil {
		return nil, err
	}

	var scanDir string
	switch l.store.Kind() {
	case store.PackageKindSkill:
		scanDir, err = agent.SkillsDir(a, l.agentPaths)
		if err != nil {
			return nil, err
		}
	default:
		home, err2 := agent.Home(a, l.agentPaths)
		if err2 != nil {
			return nil, err2
		}
		scanDir = filepath.Join(home, "mcp")
	}

	entries, err := os.ReadDir(scanDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var candidates []MigrationCandidate
	for _, e := range entries {
		id := e.Name()
		// For MCP entries the filename includes ".json"; strip it.
		if l.store.Kind() == store.PackageKindMCP {
			id = strings.TrimSuffix(id, ".json")
		}
		if _, ok := l.store.GetPackage(id); !ok {
			candidates = append(candidates, MigrationCandidate{
				ID:         id,
				SourcePath: filepath.Join(scanDir, e.Name()),
			})
		}
	}
	return candidates, nil
}

// Status returns a StatusReport for the given agent.
func (l *Linker) Status(agentName string) (StatusReport, error) {
	all, err := l.store.ListPackages()
	if err != nil {
		return StatusReport{}, err
	}

	report := StatusReport{
		Agent: agentName,
		Kind:  l.store.Kind(),
	}
	report.InstalledPackages = all

	for _, rec := range all {
		enabled := false
		for _, a := range rec.EnabledAgents {
			if a == agentName {
				enabled = true
				break
			}
		}
		if enabled {
			report.EnabledPackages = append(report.EnabledPackages, rec)
		} else {
			report.DisabledPackages = append(report.DisabledPackages, rec)
		}
	}
	return report, nil
}

// Sync recreates any missing projections tracked by the store.
func (l *Linker) Sync() error {
	links, err := l.store.ListLinks()
	if err != nil {
		return err
	}
	for _, lr := range links {
		if _, err := os.Lstat(lr.LinkPath); errors.Is(err, fs.ErrNotExist) {
			if err := l.Link(lr.PackageID, lr.Agent); err != nil {
				return err
			}
		}
	}
	return nil
}

// Doctor checks all tracked projections and returns issues for missing ones.
func (l *Linker) Doctor() ([]Issue, error) {
	links, err := l.store.ListLinks()
	if err != nil {
		return nil, err
	}
	var issues []Issue
	for _, lr := range links {
		if _, err := os.Lstat(lr.LinkPath); errors.Is(err, fs.ErrNotExist) {
			issues = append(issues, Issue{
				PackageID: lr.PackageID,
				Agent:     lr.Agent,
				Problem:   "projection missing: " + lr.LinkPath,
			})
		}
	}
	return issues, nil
}
