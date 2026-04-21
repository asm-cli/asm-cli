package linker_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/asm-cli/asm-cli/internal/installer"
	"github.com/asm-cli/asm-cli/internal/linker"
	"github.com/asm-cli/asm-cli/internal/store"
)

func setup(t *testing.T, kind store.PackageKind) (*linker.Linker, *store.Store, map[string]string, string) {
	t.Helper()
	asmHome := t.TempDir()
	agentHome := t.TempDir()
	agentPaths := map[string]string{"claude": agentHome}
	s := store.New(asmHome, kind)
	lnk := linker.New(s, agentPaths)
	return lnk, s, agentPaths, asmHome
}

func installPkg(t *testing.T, asmHome string, s *store.Store) store.PackageRecord {
	t.Helper()
	srcDir := filepath.Join(t.TempDir(), "test-pkg")
	_ = os.MkdirAll(srcDir, 0o755)
	_ = os.WriteFile(filepath.Join(srcDir, "skill.md"), []byte("# test"), 0o644)
	inst := installer.New(s, asmHome, filepath.Join(asmHome, "cache", "git"))
	rec, err := inst.Install(srcDir, installer.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

func TestLink_Skill_CreatesSymlink(t *testing.T) {
	lnk, s, agentPaths, asmHome := setup(t, store.PackageKindSkill)
	rec := installPkg(t, asmHome, s)

	if err := lnk.Link(rec.ID, "claude"); err != nil {
		t.Fatalf("Link: %v", err)
	}

	linkPath := filepath.Join(agentPaths["claude"], "skills", rec.ID)
	fi, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("link path missing: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink")
	}
}

func TestUnlink_RemovesSymlink(t *testing.T) {
	lnk, s, agentPaths, asmHome := setup(t, store.PackageKindSkill)
	rec := installPkg(t, asmHome, s)
	_ = lnk.Link(rec.ID, "claude")

	if err := lnk.Unlink(rec.ID, "claude"); err != nil {
		t.Fatalf("Unlink: %v", err)
	}
	linkPath := filepath.Join(agentPaths["claude"], "skills", rec.ID)
	if _, err := os.Lstat(linkPath); !os.IsNotExist(err) {
		t.Error("expected link to be removed")
	}
}

func TestEnable_AddsToEnabledAgents(t *testing.T) {
	lnk, s, _, asmHome := setup(t, store.PackageKindSkill)
	rec := installPkg(t, asmHome, s)

	if err := lnk.Enable(rec.ID, "claude"); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	updated, ok := s.GetPackage(rec.ID)
	if !ok {
		t.Fatal("package not found")
	}
	found := false
	for _, a := range updated.EnabledAgents {
		if a == "claude" {
			found = true
		}
	}
	if !found {
		t.Errorf("claude not in EnabledAgents: %v", updated.EnabledAgents)
	}
}

func TestDisable_RemovesFromEnabledAgents(t *testing.T) {
	lnk, s, _, asmHome := setup(t, store.PackageKindSkill)
	rec := installPkg(t, asmHome, s)
	_ = lnk.Enable(rec.ID, "claude")

	if err := lnk.Disable(rec.ID, "claude"); err != nil {
		t.Fatalf("Disable: %v", err)
	}
	updated, _ := s.GetPackage(rec.ID)
	for _, a := range updated.EnabledAgents {
		if a == "claude" {
			t.Error("claude still in EnabledAgents after disable")
		}
	}
}

func TestUse_EnablesMultipleAgents(t *testing.T) {
	asmHome := t.TempDir()
	claudeHome := t.TempDir()
	codexHome := t.TempDir()
	agentPaths := map[string]string{"claude": claudeHome, "codex": codexHome}
	s := store.New(asmHome, store.PackageKindSkill)
	lnk := linker.New(s, agentPaths)

	rec := installPkg(t, asmHome, s)
	if err := lnk.Use(rec.ID, []string{"claude", "codex"}); err != nil {
		t.Fatalf("Use: %v", err)
	}
	updated, _ := s.GetPackage(rec.ID)
	if len(updated.EnabledAgents) != 2 {
		t.Errorf("EnabledAgents = %v, want 2 agents", updated.EnabledAgents)
	}
}

func TestSync_RecreatesProjections(t *testing.T) {
	lnk, s, agentPaths, asmHome := setup(t, store.PackageKindSkill)
	rec := installPkg(t, asmHome, s)
	_ = lnk.Enable(rec.ID, "claude")

	linkPath := filepath.Join(agentPaths["claude"], "skills", rec.ID)
	_ = os.Remove(linkPath)

	if err := lnk.Sync(); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if _, err := os.Lstat(linkPath); err != nil {
		t.Error("Sync did not recreate symlink")
	}
}

func TestDoctor_DetectsBrokenLink(t *testing.T) {
	lnk, s, agentPaths, asmHome := setup(t, store.PackageKindSkill)
	rec := installPkg(t, asmHome, s)
	_ = lnk.Enable(rec.ID, "claude")

	linkPath := filepath.Join(agentPaths["claude"], "skills", rec.ID)
	_ = os.Remove(linkPath)

	issues, err := lnk.Doctor()
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) == 0 {
		t.Error("Doctor should detect the broken symlink")
	}
}

func TestStatus(t *testing.T) {
	lnk, s, _, asmHome := setup(t, store.PackageKindSkill)
	rec := installPkg(t, asmHome, s)
	_ = lnk.Enable(rec.ID, "claude")

	report, err := lnk.Status("claude")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.EnabledPackages) != 1 {
		t.Errorf("EnabledPackages = %d, want 1", len(report.EnabledPackages))
	}
}

func TestMigrate_ReturnsUnmanagedEntries(t *testing.T) {
	lnk, _, agentPaths, _ := setup(t, store.PackageKindSkill)

	skillsDir := filepath.Join(agentPaths["claude"], "skills")
	_ = os.MkdirAll(filepath.Join(skillsDir, "unmanaged-skill"), 0o755)

	candidates, err := lnk.Migrate("claude")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].ID != "unmanaged-skill" {
		t.Errorf("Migrate = %+v, want 1 candidate 'unmanaged-skill'", candidates)
	}
}
