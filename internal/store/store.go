package store

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

type storeState struct {
	Packages map[string]PackageRecord `json:"packages"`
	Links    []LinkRecord             `json:"links"`
}

type Store struct {
	asmHome string
	kind    PackageKind
}

func New(asmHome string, kind PackageKind) *Store {
	return &Store{asmHome: asmHome, kind: kind}
}

// Kind returns the PackageKind this store manages.
func (s *Store) Kind() PackageKind { return s.kind }

func (s *Store) statePath() string {
	return filepath.Join(s.asmHome, "store", string(s.kind)+"s", "state.json")
}

func (s *Store) load() (storeState, error) {
	st := storeState{Packages: make(map[string]PackageRecord)}
	data, err := os.ReadFile(s.statePath())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return st, nil
		}
		return st, err
	}
	if err := json.Unmarshal(data, &st); err != nil {
		return st, err
	}
	if st.Packages == nil {
		st.Packages = make(map[string]PackageRecord)
	}
	return st, nil
}

func (s *Store) save(st storeState) error {
	if err := os.MkdirAll(filepath.Dir(s.statePath()), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.statePath(), data, 0o644)
}

func (s *Store) GetPackage(id string) (PackageRecord, bool) {
	st, err := s.load()
	if err != nil {
		return PackageRecord{}, false
	}
	r, ok := st.Packages[id]
	return r, ok
}

func (s *Store) SavePackage(r PackageRecord) error {
	st, err := s.load()
	if err != nil {
		return err
	}
	st.Packages[r.ID] = r
	return s.save(st)
}

func (s *Store) DeletePackage(id string) error {
	st, err := s.load()
	if err != nil {
		return err
	}
	delete(st.Packages, id)
	return s.save(st)
}

func (s *Store) ListPackages() ([]PackageRecord, error) {
	st, err := s.load()
	if err != nil {
		return nil, err
	}
	out := make([]PackageRecord, 0, len(st.Packages))
	for _, r := range st.Packages {
		out = append(out, r)
	}
	return out, nil
}

func (s *Store) GetLinks(packageID string) ([]LinkRecord, error) {
	st, err := s.load()
	if err != nil {
		return nil, err
	}
	var out []LinkRecord
	for _, lr := range st.Links {
		if lr.PackageID == packageID {
			out = append(out, lr)
		}
	}
	return out, nil
}

func (s *Store) SaveLink(r LinkRecord) error {
	st, err := s.load()
	if err != nil {
		return err
	}
	for i, lr := range st.Links {
		if lr.PackageID == r.PackageID && lr.Agent == r.Agent {
			st.Links[i] = r
			return s.save(st)
		}
	}
	st.Links = append(st.Links, r)
	return s.save(st)
}

func (s *Store) DeleteLink(packageID, agent string) error {
	st, err := s.load()
	if err != nil {
		return err
	}
	var filtered []LinkRecord
	for _, lr := range st.Links {
		if !(lr.PackageID == packageID && lr.Agent == agent) {
			filtered = append(filtered, lr)
		}
	}
	st.Links = filtered
	return s.save(st)
}

func (s *Store) ListLinks() ([]LinkRecord, error) {
	st, err := s.load()
	if err != nil {
		return nil, err
	}
	return st.Links, nil
}
