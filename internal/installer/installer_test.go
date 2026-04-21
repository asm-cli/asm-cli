package installer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/asm-cli/asm-cli/internal/installer"
	"github.com/asm-cli/asm-cli/internal/store"
)

func newInstaller(t *testing.T, kind store.PackageKind) (*installer.Installer, *store.Store, string) {
	t.Helper()
	asmHome := t.TempDir()
	s := store.New(asmHome, kind)
	inst := installer.New(s, asmHome, filepath.Join(asmHome, "cache", "git"))
	return inst, s, asmHome
}

func makeLocalPkg(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skill.md"), []byte("# "+name), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestInstall_Local(t *testing.T) {
	inst, s, _ := newInstaller(t, store.PackageKindSkill)
	src := makeLocalPkg(t, "my-skill")

	rec, err := inst.Install(src, installer.Options{})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if rec.ID != "my-skill" {
		t.Errorf("ID = %q, want my-skill", rec.ID)
	}
	if rec.Source.Type != store.SourceTypeLocal {
		t.Errorf("SourceType = %q", rec.Source.Type)
	}
	if _, ok := s.GetPackage("my-skill"); !ok {
		t.Error("package not persisted to store")
	}
	if _, err := os.Stat(rec.StorePath); err != nil {
		t.Errorf("store path missing: %v", err)
	}
}

func TestInstall_Local_CustomID(t *testing.T) {
	inst, s, _ := newInstaller(t, store.PackageKindSkill)
	src := makeLocalPkg(t, "original-name")

	rec, err := inst.Install(src, installer.Options{ID: "custom-id"})
	if err != nil {
		t.Fatal(err)
	}
	if rec.ID != "custom-id" {
		t.Errorf("ID = %q, want custom-id", rec.ID)
	}
	if _, ok := s.GetPackage("custom-id"); !ok {
		t.Error("package not persisted with custom ID")
	}
}

func TestInstall_AlreadyExists(t *testing.T) {
	inst, _, _ := newInstaller(t, store.PackageKindSkill)
	src := makeLocalPkg(t, "my-skill")

	if _, err := inst.Install(src, installer.Options{}); err != nil {
		t.Fatal(err)
	}
	_, err := inst.Install(src, installer.Options{})
	if err == nil {
		t.Fatal("expected error on duplicate install")
	}
}

func TestUninstall(t *testing.T) {
	inst, s, _ := newInstaller(t, store.PackageKindSkill)
	src := makeLocalPkg(t, "my-skill")
	if _, err := inst.Install(src, installer.Options{}); err != nil {
		t.Fatal(err)
	}
	if err := inst.Uninstall("my-skill"); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if _, ok := s.GetPackage("my-skill"); ok {
		t.Error("package still in store after uninstall")
	}
}

func TestUninstall_NotFound(t *testing.T) {
	inst, _, _ := newInstaller(t, store.PackageKindSkill)
	err := inst.Uninstall("ghost")
	if err == nil {
		t.Error("expected error for missing package")
	}
}

func TestUpdate_Local(t *testing.T) {
	inst, _, _ := newInstaller(t, store.PackageKindSkill)
	src := makeLocalPkg(t, "my-skill")
	if _, err := inst.Install(src, installer.Options{}); err != nil {
		t.Fatal(err)
	}
	// Add a new file to the source
	_ = os.WriteFile(filepath.Join(src, "new.md"), []byte("update"), 0o644)

	if err := inst.Update("my-skill"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	rec, _ := inst.Store().GetPackage("my-skill")
	if _, err := os.Stat(filepath.Join(rec.StorePath, "new.md")); err != nil {
		t.Error("updated file not in store path")
	}
}
