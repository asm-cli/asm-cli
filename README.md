# ASM — Agent Skills Manager

Manage skills, MCP servers, and plugins for AI agents from a single store.

[中文文档](README.zh.md)

---

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Quickstart](#quickstart)
- [How It Works](#how-it-works)
- [Configuration](#configuration)
- [Init](#init)
- [Skills Commands](#skills-commands)
- [MCP Commands](#mcp-commands)
- [Plugins Commands](#plugins-commands)
- [Global Flags](#global-flags)
- [Directory Layout](#directory-layout)

---

## Overview

ASM maintains a central store under `~/.asm/` and projects packages into each agent's home directory:

| Kind | Store | Projection |
|------|-------|------------|
| **Skills** | `~/.asm/store/skills/<id>` | symlink → `~/.claude/skills/<id>` |
| **MCP servers** | `~/.asm/store/mcps/<id>/config.json` | injected into agent's native config (`config.toml` / `settings.json`) |
| **Plugins** | `~/.asm/store/plugins/<id>` | symlink → agent plugin directory |

Install once, enable for any number of agents — no duplication.

---

## Installation

```bash
git clone https://github.com/6xiaowu9/asm
cd asm
go build -o ~/.local/bin/asm .
```

Requires Go 1.22+.

---

## Quickstart

Initialize ASM first. This creates `~/.asm/`, writes the default config, and
injects the `@ASM.md` reference into detected agent configs.

```bash
asm init
```

### Use from a shell

Install a skill, MCP server, or plugin once, then enable it for one or more
agents.

```bash
asm skills install https://github.com/user/my-skill --agents claude,codex
asm mcp install https://github.com/user/my-mcp --agents codex
asm plugins install https://github.com/user/my-plugin --agents claude,codex
```

Check what ASM manages and whether projections are healthy.

```bash
asm skills list
asm skills status --agent codex
asm skills doctor
```

If a package already exists in the ASM store, enable or use it instead of
installing it again.

```bash
asm skills enable my-skill --agent claude
asm skills use my-skill --agents claude,codex
```

### Use from an AI CLI

After `asm init`, supported AI CLIs receive the `@ASM.md` instructions. Ask the
agent to use ASM package commands directly, or use the slash-command mapping
when the CLI supports it.

```text
/skills install https://github.com/user/my-skill --agents claude,codex
/mcp install https://github.com/user/my-mcp --agents codex
/plugins install https://github.com/user/my-plugin --agents claude,codex
```

These map to the same ASM commands:

```text
/skills   -> asm skills
/mcp      -> asm mcp
/plugins  -> asm plugins
```

For example, in an AI CLI prompt:

```text
Use ASM to install https://github.com/user/my-skill for claude and codex.
```

---

## How It Works

```
Source (local path / git URL)
        │
        ▼
   ASM Store  (~/.asm/store/{skills,mcps,plugins}/<id>)
        │
        ├── symlink ──────────────► ~/.claude/skills/<id>
        ├── symlink ──────────────► ~/.codex/skills/<id>
        ├── config injection ─────► ~/.codex/config.toml
        └── config injection ─────► ~/.claude/settings.json
```

- **install** copies files into the store
- **enable / use** creates the projection in the agent directory
- **migrate** imports existing unmanaged packages into the store
- **sync** recreates any broken projections

---

## Configuration

ASM reads `~/.asm/config.toml`. Defaults are used when absent.

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

`ASM_HOME` environment variable overrides `asm_home` at runtime.

---

## Init

Initialize ASM and inject the `@ASM.md` reference into all configured agents.

```bash
asm init
```

`asm init` is idempotent — safe to run multiple times.

---

## Skills Commands

All commands share the `asm skills` prefix.

### install

```bash
asm skills install <source> [flags]
```

| Flag | Description |
|------|-------------|
| `--id <string>` | Override the package ID |
| `--subdir <path>` | Subdirectory within the source to install |
| `--ref <string>` | Git ref (branch, tag, or commit SHA) |
| `--agents <list>` | Comma-separated agents to enable after install |

```bash
asm skills install ./my-skill
asm skills install https://github.com/user/my-skill --agents claude,codex
asm skills install https://github.com/user/repo --subdir skills/my-skill --ref v1.2.0
```

### list

```bash
asm skills list
```

### status

```bash
asm skills status [--agent <name>]
```

### enable / disable

```bash
asm skills enable <id> [--agent <name>]
asm skills disable <id> [--agent <name>]
```

### use

Enable for multiple agents at once.

```bash
asm skills use <id> --agents claude,codex
```

### update

```bash
asm skills update <id>
asm skills update --all
```

### remove

Remove projections, store record, and stored files.

```bash
asm skills remove <id>
```

### sync

Recreate projections that are tracked but missing from disk.

```bash
asm skills sync
```

### doctor

Report broken projections.

```bash
asm skills doctor
```

### migrate

Import unmanaged skills from an agent directory into the store.

```bash
asm skills migrate [--agent <name>]
```

---

## MCP Commands

`asm mcp` exposes the same subcommands. MCP packages are stored under `~/.asm/store/mcps/<id>` and projected by injecting entries into the agent's native config file (`~/.codex/config.toml` or `~/.claude/settings.json`).

```bash
asm mcp install <source> [flags]
asm mcp list
asm mcp status  [--agent <name>]
asm mcp enable  <id> [--agent <name>]
asm mcp disable <id> [--agent <name>]
asm mcp use     <id> [--agents <list>]
asm mcp update  <id> | --all
asm mcp remove  <id>
asm mcp sync
asm mcp doctor
asm mcp migrate [--agent <name>]
```

`migrate` reads the agent's native MCP config, imports each server into the store, rewrites the `command` path to the ASM store copy, and removes any local binaries from the agent home.

---

## Plugins Commands

`asm plugins` manages agent-native plugin extensions (e.g. superpowers). Plugins are stored under `~/.asm/store/plugins/<id>` and symlinked into each agent's plugin directory.

```bash
asm plugins install <source> [flags]
asm plugins list
asm plugins status  [--agent <name>]
asm plugins enable  <id> [--agent <name>]
asm plugins disable <id> [--agent <name>]
asm plugins use     <id> [--agents <list>]
asm plugins update  <id> | --all
asm plugins remove  <id>
asm plugins sync
asm plugins doctor
asm plugins migrate [--agent <name>]
```

---

## Global Flags

| Flag | Description |
|------|-------------|
| `--asm-home <path>` | Override the ASM home directory |
| `-h, --help` | Help for any command |

Resolution order for ASM home: `--asm-home` flag → `ASM_HOME` env → `~/.asm`

---

## Directory Layout

```
~/.asm/
├── config.toml
├── ASM.md
├── cache/
│   └── git/
└── store/
    ├── skills/
    │   ├── state.json
    │   └── <id>/
    ├── mcps/
    │   ├── state.json
    │   └── <id>/
    │       └── config.json
    └── plugins/
        ├── state.json
        └── <id>/

~/.claude/
└── skills/
    └── <id> -> ~/.asm/store/skills/<id>    # symlink

~/.codex/
├── config.toml   # MCP entries injected here
└── skills/
    └── <id> -> ~/.asm/store/skills/<id>    # symlink
```
