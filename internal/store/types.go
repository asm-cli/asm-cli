package store

import (
	"fmt"
	"time"
)

type PackageKind string

const (
	PackageKindSkill PackageKind = "skill"
	PackageKindMCP   PackageKind = "mcp"
)

type SourceType string

const (
	SourceTypeLocal SourceType = "local"
	SourceTypeGit   SourceType = "git"
)

type InstallSource struct {
	Type   SourceType `json:"type"`
	Ref    string     `json:"ref"`
	Subdir string     `json:"subdir,omitempty"`
}

type PackageRecord struct {
	ID            string        `json:"id"`
	Kind          PackageKind   `json:"kind"`
	Source        InstallSource `json:"source"`
	Revision      string        `json:"revision"`
	InstalledAt   time.Time     `json:"installed_at"`
	EnabledAgents []string      `json:"enabled_agents"`
	EntryPoints   []string      `json:"entry_points"`
	StorePath     string        `json:"store_path"`
}

type LinkRecord struct {
	PackageID string      `json:"package_id"`
	Agent     string      `json:"agent"`
	Kind      PackageKind `json:"kind"`
	Mode      string      `json:"mode"`
	LinkPath  string      `json:"link_path"`
	Target    string      `json:"target"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type NotFoundError struct{ ID string }

func (e *NotFoundError) Error() string { return fmt.Sprintf("package %q not found", e.ID) }

type AlreadyExistsError struct{ ID string }

func (e *AlreadyExistsError) Error() string {
	return fmt.Sprintf("package %q already installed", e.ID)
}
