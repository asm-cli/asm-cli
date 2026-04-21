package installer

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/asm-cli/asm-cli/internal/store"
)

// Options controls how a package is installed.
type Options struct {
	ID     string
	Subdir string
	Ref    string
	Git    bool
}

// Installer manages package installation into the ASM store.
type Installer struct {
	store       *store.Store
	asmHome     string
	gitCacheDir string
}

// New creates a new Installer.
func New(s *store.Store, asmHome, gitCacheDir string) *Installer {
	return &Installer{store: s, asmHome: asmHome, gitCacheDir: gitCacheDir}
}

// Store exposes the underlying store (useful for tests).
func (i *Installer) Store() *store.Store { return i.store }

// Install installs a package from a local path or git URL.
func (i *Installer) Install(source string, opts Options) (store.PackageRecord, error) {
	if opts.Git || isGitURL(source) {
		return i.installGit(source, opts)
	}
	return i.installLocal(source, opts)
}

func (i *Installer) installLocal(source string, opts Options) (store.PackageRecord, error) {
	abs, err := filepath.Abs(source)
	if err != nil {
		return store.PackageRecord{}, fmt.Errorf("resolve path: %w", err)
	}

	srcDir := abs
	if opts.Subdir != "" {
		srcDir = filepath.Join(abs, opts.Subdir)
	}

	id := opts.ID
	if id == "" {
		id = filepath.Base(abs)
	}

	if _, ok := i.store.GetPackage(id); ok {
		return store.PackageRecord{}, &store.AlreadyExistsError{ID: id}
	}

	storePath := i.storePath(id)
	if err := os.MkdirAll(filepath.Dir(storePath), 0o755); err != nil {
		return store.PackageRecord{}, fmt.Errorf("create store dir: %w", err)
	}

	if err := copyDir(srcDir, storePath); err != nil {
		return store.PackageRecord{}, fmt.Errorf("copy dir: %w", err)
	}

	rev, err := dirRevision(storePath)
	if err != nil {
		return store.PackageRecord{}, fmt.Errorf("compute revision: %w", err)
	}

	rec := store.PackageRecord{
		ID:   id,
		Kind: i.store.Kind(),
		Source: store.InstallSource{
			Type:   store.SourceTypeLocal,
			Ref:    abs, // store original source path for updates
			Subdir: opts.Subdir,
		},
		Revision:    rev,
		InstalledAt: time.Now(),
		StorePath:   storePath,
	}

	if err := i.store.SavePackage(rec); err != nil {
		return store.PackageRecord{}, fmt.Errorf("save package: %w", err)
	}

	return rec, nil
}

func (i *Installer) installGit(source string, opts Options) (store.PackageRecord, error) {
	id := opts.ID
	if id == "" {
		id = gitRepoID(source)
	}

	if _, ok := i.store.GetPackage(id); ok {
		return store.PackageRecord{}, &store.AlreadyExistsError{ID: id}
	}

	cacheDir := filepath.Join(i.gitCacheDir, sanitizePath(id))

	if _, err := os.Stat(cacheDir); errors.Is(err, fs.ErrNotExist) {
		// Clone
		if err := os.MkdirAll(filepath.Dir(cacheDir), 0o755); err != nil {
			return store.PackageRecord{}, fmt.Errorf("create cache dir: %w", err)
		}
		if err := gitRun("clone", source, cacheDir); err != nil {
			return store.PackageRecord{}, fmt.Errorf("git clone: %w", err)
		}
	} else {
		// Pull
		if err := gitRunIn(cacheDir, "pull"); err != nil {
			return store.PackageRecord{}, fmt.Errorf("git pull: %w", err)
		}
	}

	if opts.Ref != "" {
		if err := gitRunIn(cacheDir, "checkout", opts.Ref); err != nil {
			return store.PackageRecord{}, fmt.Errorf("git checkout: %w", err)
		}
	}

	srcDir := cacheDir
	if opts.Subdir != "" {
		srcDir = filepath.Join(cacheDir, opts.Subdir)
	}

	storePath := i.storePath(id)
	if err := os.MkdirAll(filepath.Dir(storePath), 0o755); err != nil {
		return store.PackageRecord{}, fmt.Errorf("create store dir: %w", err)
	}

	if err := copyDir(srcDir, storePath); err != nil {
		return store.PackageRecord{}, fmt.Errorf("copy dir: %w", err)
	}

	rev, err := gitRevision(cacheDir)
	if err != nil {
		return store.PackageRecord{}, fmt.Errorf("git revision: %w", err)
	}

	rec := store.PackageRecord{
		ID:   id,
		Kind: i.store.Kind(),
		Source: store.InstallSource{
			Type:   store.SourceTypeGit,
			Ref:    source, // store clone URL so Update can re-clone if cache is missing
			Subdir: opts.Subdir,
		},
		Revision:    rev,
		InstalledAt: time.Now(),
		StorePath:   storePath,
	}

	if err := i.store.SavePackage(rec); err != nil {
		return store.PackageRecord{}, fmt.Errorf("save package: %w", err)
	}

	return rec, nil
}

// InstallMCPConfig writes a raw JSON config (and optionally a binary) to the
// store and registers the package. Used when migrating MCP servers from agent
// config files.
//
// If localBinary is non-empty, the file is copied into storePath/ and the
// "command" field in configJSON is rewritten to the store copy's path so that
// subsequent inject operations use the ASM-managed binary.
func (i *Installer) InstallMCPConfig(id string, configJSON []byte, localBinary string) (store.PackageRecord, error) {
	if _, ok := i.store.GetPackage(id); ok {
		return store.PackageRecord{}, &store.AlreadyExistsError{ID: id}
	}

	storePath := i.storePath(id)
	if err := os.MkdirAll(storePath, 0o755); err != nil {
		return store.PackageRecord{}, fmt.Errorf("create store dir: %w", err)
	}

	// Copy local binary into the store and rewrite the command path.
	if localBinary != "" {
		binName := filepath.Base(localBinary)
		storeBin := filepath.Join(storePath, binName)
		if err := copyFile(localBinary, storeBin); err != nil {
			return store.PackageRecord{}, fmt.Errorf("copy binary: %w", err)
		}
		// Preserve executable bit.
		if info, err := os.Stat(localBinary); err == nil {
			_ = os.Chmod(storeBin, info.Mode())
		}
		// Rewrite "command" in configJSON to point to the store copy.
		var cfg map[string]interface{}
		if err := json.Unmarshal(configJSON, &cfg); err == nil {
			cfg["command"] = storeBin
			if updated, err := json.Marshal(cfg); err == nil {
				configJSON = updated
			}
		}
	}

	cfgFile := filepath.Join(storePath, "config.json")
	if err := os.WriteFile(cfgFile, configJSON, 0o644); err != nil {
		return store.PackageRecord{}, fmt.Errorf("write config: %w", err)
	}

	rev, _ := dirRevision(storePath)
	rec := store.PackageRecord{
		ID:          id,
		Kind:        i.store.Kind(),
		Source:      store.InstallSource{Type: store.SourceTypeLocal},
		Revision:    rev,
		InstalledAt: time.Now(),
		StorePath:   storePath,
	}
	if err := i.store.SavePackage(rec); err != nil {
		return store.PackageRecord{}, fmt.Errorf("save package: %w", err)
	}
	return rec, nil
}

// Uninstall removes a package from the store and deletes its store path,
// but does not remove any links.
func (i *Installer) Uninstall(id string) error {
	rec, ok := i.store.GetPackage(id)
	if !ok {
		return &store.NotFoundError{ID: id}
	}

	if err := i.store.DeletePackage(id); err != nil {
		return fmt.Errorf("delete package record: %w", err)
	}

	if err := os.RemoveAll(rec.StorePath); err != nil {
		return fmt.Errorf("remove store path: %w", err)
	}

	return nil
}

// Remove removes all links for the package from disk and the store, then calls Uninstall.
func (i *Installer) Remove(id string) error {
	links, err := i.store.GetLinks(id)
	if err != nil {
		return fmt.Errorf("get links: %w", err)
	}

	for _, lr := range links {
		if err := os.Remove(lr.LinkPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("remove link %s: %w", lr.LinkPath, err)
		}
		if err := i.store.DeleteLink(lr.PackageID, lr.Agent); err != nil {
			return fmt.Errorf("delete link record: %w", err)
		}
	}

	return i.Uninstall(id)
}

// Update re-copies or re-pulls a package and updates its revision.
func (i *Installer) Update(id string) error {
	rec, ok := i.store.GetPackage(id)
	if !ok {
		return &store.NotFoundError{ID: id}
	}

	if rec.Source.Type == store.SourceTypeGit {
		return i.updateGit(rec)
	}
	return i.updateLocal(rec)
}

func (i *Installer) updateLocal(rec store.PackageRecord) error {
	srcDir := rec.Source.Ref // for local, Ref holds the original absolute source path
	// If Ref is empty (older records), we can't update — nothing to do
	// The source path is stored in Ref field for local installs
	// Actually, looking at installLocal we don't store source path in Ref.
	// We need to store the source path. Let's use Source.Ref for local type.
	// Since we set it during install, we need to check if it's set.
	if srcDir == "" {
		return fmt.Errorf("local source path not recorded for %q; cannot update", rec.ID)
	}

	if rec.Source.Subdir != "" {
		srcDir = filepath.Join(srcDir, rec.Source.Subdir)
	}

	if err := os.RemoveAll(rec.StorePath); err != nil {
		return fmt.Errorf("remove store path: %w", err)
	}

	if err := copyDir(srcDir, rec.StorePath); err != nil {
		return fmt.Errorf("copy dir: %w", err)
	}

	rev, err := dirRevision(rec.StorePath)
	if err != nil {
		return fmt.Errorf("compute revision: %w", err)
	}

	rec.Revision = rev
	return i.store.SavePackage(rec)
}

func (i *Installer) updateGit(rec store.PackageRecord) error {
	cacheDir := filepath.Join(i.gitCacheDir, sanitizePath(rec.ID))

	if _, err := os.Stat(cacheDir); errors.Is(err, fs.ErrNotExist) {
		// Cache missing — re-clone from stored URL
		if err := os.MkdirAll(filepath.Dir(cacheDir), 0o755); err != nil {
			return fmt.Errorf("create cache dir: %w", err)
		}
		if err := gitRun("clone", rec.Source.Ref, cacheDir); err != nil {
			return fmt.Errorf("git clone: %w", err)
		}
	} else {
		if err := gitRunIn(cacheDir, "pull"); err != nil {
			return fmt.Errorf("git pull: %w", err)
		}
	}

	srcDir := cacheDir
	if rec.Source.Subdir != "" {
		srcDir = filepath.Join(cacheDir, rec.Source.Subdir)
	}

	if err := os.RemoveAll(rec.StorePath); err != nil {
		return fmt.Errorf("remove store path: %w", err)
	}

	if err := copyDir(srcDir, rec.StorePath); err != nil {
		return fmt.Errorf("copy dir: %w", err)
	}

	rev, err := gitRevision(cacheDir)
	if err != nil {
		return fmt.Errorf("git revision: %w", err)
	}

	rec.Revision = rev
	return i.store.SavePackage(rec)
}

// UpdateAll updates all packages managed by this installer's store.
func (i *Installer) UpdateAll() error {
	pkgs, err := i.store.ListPackages()
	if err != nil {
		return err
	}
	for _, pkg := range pkgs {
		if err := i.Update(pkg.ID); err != nil {
			return fmt.Errorf("update %s: %w", pkg.ID, err)
		}
	}
	return nil
}

// storePath returns the path where a package should be stored.
func (i *Installer) storePath(id string) string {
	return i.store.PackagePath(id)
}

// --- Helper functions ---

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

func dirRevision(dir string) (string, error) {
	h := sha256.New()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "%s\n", rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		h.Write(data)
		return nil
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:12], nil
}

func gitRevision(dir string) (string, error) {
	return gitOutput(dir, "rev-parse", "--short", "HEAD")
}

func isGitURL(s string) bool {
	return strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "git@") ||
		strings.HasPrefix(s, "git://") ||
		strings.HasPrefix(s, "ssh://")
}

func gitRepoID(url string) string {
	base := filepath.Base(url)
	return strings.TrimSuffix(base, ".git")
}

func sanitizePath(s string) string {
	r := strings.NewReplacer("/", "-", ":", "-", "@", "-")
	return r.Replace(s)
}

func gitRun(args ...string) error {
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func gitRunIn(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%w\n%s", err, strings.TrimSpace(string(ee.Stderr)))
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
