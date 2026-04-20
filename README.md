# ASM — Agent Skills Manager

A CLI tool for managing AI agent skill packages and MCP (Model Context Protocol) packages across multiple agents (Claude, Codex, Cursor, Gemini).

[中文文档](README.zh.md)

---

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [How It Works](#how-it-works)
- [Configuration](#configuration)
- [Skills Commands](#skills-commands)
- [MCP Commands](#mcp-commands)
- [Global Flags](#global-flags)
- [Directory Layout](#directory-layout)

---

## Overview

ASM maintains a central store of packages under `~/.asm/` and projects them into each agent's home directory:

- **Skills** are projected as symlinks (`~/.claude/skills/<id>` → `~/.asm/store/skills/<id>`)
- **MCP packages** are projected as JSON manifests (`~/.claude/mcp/<id>.json`)

This lets you install once and enable the same package for multiple agents without duplication.

---

## Installation

```bash
git clone https://github.com/6xiaowu9/asm
cd asm
go build -o ~/.local/bin/asm .
```

Requires Go 1.22+.

---

## How It Works

```
Source (local path / git URL)
        │
        ▼
   ASM Store  (~/.asm/store/skills/<id>  or  ~/.asm/store/mcps/<id>)
        │
        ├─── symlink / JSON manifest ──► ~/.claude/skills/<id>
        ├─── symlink / JSON manifest ──► ~/.codex/skills/<id>
        └─── ...
```

**Install** copies files into the store. **Enable / use** creates the projection in the agent directory. **Sync** recreates any broken projections.

---

## Configuration

ASM looks for a config file at `~/.asm/config.toml`. If absent, defaults are used.

```toml
asm_home      = "/home/you/.asm"
link_mode     = "auto"
git_cache_dir = "/home/you/.asm/cache/git"
default_agents = ["claude"]

[agent_paths]
claude = "/home/you/.claude"
codex  = "/home/you/.codex"
cursor = "/home/you/.cursor"
gemini = "/home/you/.gemini"
```

The `ASM_HOME` environment variable overrides `asm_home` at runtime.

---

## Skills Commands

All commands share the `asm skills` prefix.

### install

Install a skill from a local directory or git URL.

```bash
asm skills install <source> [flags]
```

| Flag | Description |
|------|-------------|
| `--id <string>` | Override the package ID (defaults to directory/repo name) |
| `--subdir <path>` | Subdirectory within the source to install |
| `--ref <string>` | Git ref to check out (branch, tag, or commit SHA) |
| `--agents <list>` | Comma-separated agents to enable immediately after install |

**Examples:**

```bash
# Install from a local directory
asm skills install ./my-skill

# Install from git and immediately enable for claude
asm skills install https://github.com/user/my-skill --agents claude

# Install a subdirectory from a git repo at a specific tag
asm skills install https://github.com/user/repo --subdir skills/my-skill --ref v1.2.0

# Install with a custom ID
asm skills install ./my-skill --id my-custom-id
```

### list

List all installed skill packages.

```bash
asm skills list
```

### status

Show which skills are enabled or disabled for an agent.

```bash
asm skills status [--agent <name>]
```

Defaults to the first agent in `default_agents` (or `claude`).

### enable

Create the projection and mark the skill as enabled for an agent.

```bash
asm skills enable <id> [--agent <name>]
```

### disable

Remove the projection and mark the skill as disabled for an agent.

```bash
asm skills disable <id> [--agent <name>]
```

### use

Enable a skill for one or more agents at once.

```bash
asm skills use <id> [--agents <list>]
```

```bash
asm skills use my-skill --agents claude,codex
```

### link

Create the projection without changing the enabled/disabled state in the store.

```bash
asm skills link <id> [--agent <name>]
```

### unlink

Remove the projection without changing the enabled/disabled state in the store.

```bash
asm skills unlink <id> [--agent <name>]
```

### update

Re-copy (local) or re-pull (git) a package to pick up changes.

```bash
asm skills update <id>
asm skills update --all
```

### uninstall

Remove the store record and files. Existing projections (symlinks) are **not** removed.

```bash
asm skills uninstall <id>
```

### remove

Remove all projections, the store record, and the stored files.

```bash
asm skills remove <id>
```

### sync

Recreate any projections that are tracked in the store but missing from disk (e.g. after moving the agent home directory).

```bash
asm skills sync
```

### doctor

Inspect all tracked projections and report broken ones.

```bash
asm skills doctor
```

### migrate

Scan an agent's skills directory for entries not yet tracked by ASM.

```bash
asm skills migrate [--agent <name>]
```

---

## MCP Commands

`asm mcp` exposes the same set of subcommands as `asm skills`. The difference is the package kind and projection format:

- MCP packages are stored under `~/.asm/store/mcps/<id>`
- Projections are JSON manifests at `~/.claude/mcp/<id>.json`

```bash
asm mcp install <source> [flags]
asm mcp list
asm mcp status [--agent <name>]
asm mcp enable <id> [--agent <name>]
asm mcp disable <id> [--agent <name>]
asm mcp use <id> [--agents <list>]
asm mcp update <id> | --all
asm mcp remove <id>
asm mcp sync
asm mcp doctor
asm mcp migrate [--agent <name>]
```

---

## Global Flags

| Flag | Description |
|------|-------------|
| `--asm-home <path>` | Override the ASM home directory for this invocation |
| `-h, --help` | Help for any command |

The resolution order for ASM home:

1. `--asm-home` flag
2. `ASM_HOME` environment variable
3. `~/.asm`

---

## Directory Layout

```
~/.asm/
├── config.toml
├── cache/
│   └── git/                  # git clone cache
└── store/
    ├── skills/
    │   ├── state.json        # package and link records
    │   └── <id>/             # copied skill files
    └── mcps/
        ├── state.json
        └── <id>/             # copied MCP files

~/.claude/
├── skills/
│   └── <id> -> ~/.asm/store/skills/<id>   # symlink
└── mcp/
    └── <id>.json             # generated manifest
```
