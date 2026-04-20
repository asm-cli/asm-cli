package store_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/6xiaowu9/asm/internal/store"
)

func TestStore_SaveAndGetPackage(t *testing.T) {
	dir := t.TempDir()
	s := store.New(dir, store.PackageKindSkill)

	rec := store.PackageRecord{
		ID:          "my-skill",
		Kind:        store.PackageKindSkill,
		Source:      store.InstallSource{Type: store.SourceTypeLocal, Ref: "/tmp/my-skill"},
		Revision:    "abc123",
		InstalledAt: time.Now().UTC().Truncate(time.Second),
		StorePath:   filepath.Join(dir, "store", "skills", "my-skill"),
	}

	if err := s.SavePackage(rec); err != nil {
		t.Fatalf("SavePackage: %v", err)
	}

	got, ok := s.GetPackage("my-skill")
	if !ok {
		t.Fatal("GetPackage: not found after save")
	}
	if got.ID != rec.ID || got.Revision != rec.Revision {
		t.Errorf("got %+v, want %+v", got, rec)
	}
}

func TestStore_GetPackage_NotFound(t *testing.T) {
	s := store.New(t.TempDir(), store.PackageKindSkill)
	_, ok := s.GetPackage("missing")
	if ok {
		t.Fatal("expected not found")
	}
}

func TestStore_DeletePackage(t *testing.T) {
	dir := t.TempDir()
	s := store.New(dir, store.PackageKindSkill)
	rec := store.PackageRecord{ID: "pkg", Kind: store.PackageKindSkill,
		Source:    store.InstallSource{Type: store.SourceTypeLocal, Ref: "/tmp/pkg"},
		StorePath: filepath.Join(dir, "store", "skills", "pkg"),
	}
	if err := s.SavePackage(rec); err != nil {
		t.Fatal(err)
	}
	if err := s.DeletePackage("pkg"); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.GetPackage("pkg"); ok {
		t.Fatal("still present after delete")
	}
}

func TestStore_ListPackages(t *testing.T) {
	dir := t.TempDir()
	s := store.New(dir, store.PackageKindSkill)
	for _, id := range []string{"a", "b", "c"} {
		_ = s.SavePackage(store.PackageRecord{ID: id, Kind: store.PackageKindSkill,
			Source:    store.InstallSource{Type: store.SourceTypeLocal, Ref: "/tmp/" + id},
			StorePath: filepath.Join(dir, "store", "skills", id),
		})
	}
	pkgs, err := s.ListPackages()
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 3 {
		t.Fatalf("want 3, got %d", len(pkgs))
	}
}

func TestStore_SaveAndListLinks(t *testing.T) {
	dir := t.TempDir()
	s := store.New(dir, store.PackageKindSkill)

	lr := store.LinkRecord{
		PackageID: "my-skill",
		Agent:     "claude",
		Kind:      store.PackageKindSkill,
		Mode:      "symlink",
		LinkPath:  "/home/user/.claude/skills/my-skill",
		Target:    filepath.Join(dir, "store", "skills", "my-skill"),
		UpdatedAt: time.Now().UTC().Truncate(time.Second),
	}
	if err := s.SaveLink(lr); err != nil {
		t.Fatal(err)
	}
	links, err := s.ListLinks()
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 1 || links[0].PackageID != "my-skill" {
		t.Fatalf("unexpected links: %+v", links)
	}
}

func TestStore_DeleteLink(t *testing.T) {
	dir := t.TempDir()
	s := store.New(dir, store.PackageKindSkill)
	lr := store.LinkRecord{PackageID: "pkg", Agent: "claude", Kind: store.PackageKindSkill}
	_ = s.SaveLink(lr)
	if err := s.DeleteLink("pkg", "claude"); err != nil {
		t.Fatal(err)
	}
	links, _ := s.ListLinks()
	if len(links) != 0 {
		t.Fatal("link still present after delete")
	}
}

func TestStore_GetLinks(t *testing.T) {
	dir := t.TempDir()
	s := store.New(dir, store.PackageKindSkill)
	_ = s.SaveLink(store.LinkRecord{PackageID: "pkg", Agent: "claude", Kind: store.PackageKindSkill})
	_ = s.SaveLink(store.LinkRecord{PackageID: "pkg", Agent: "codex", Kind: store.PackageKindSkill})
	_ = s.SaveLink(store.LinkRecord{PackageID: "other", Agent: "claude", Kind: store.PackageKindSkill})

	links, err := s.GetLinks("pkg")
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 2 {
		t.Fatalf("want 2 links for pkg, got %d", len(links))
	}
}
