# ASM — AI Agent 包管理器

为 AI Agent 统一管理 Skills、MCP 服务器和插件。

[English](README.md)

---

## 目录

- [概述](#概述)
- [安装](#安装)
- [工作原理](#工作原理)
- [配置](#配置)
- [初始化](#初始化)
- [Skills 命令](#skills-命令)
- [MCP 命令](#mcp-命令)
- [Plugins 命令](#plugins-命令)
- [全局参数](#全局参数)
- [目录结构](#目录结构)

---

## 概述

ASM 在 `~/.asm/` 下维护一个集中式包存储，并将包投影到各 Agent 的主目录：

| 类型 | 存储路径 | 投影方式 |
|------|---------|---------|
| **Skills** | `~/.asm/store/skills/<id>` | 符号链接 → `~/.claude/skills/<id>` |
| **MCP 服务器** | `~/.asm/store/mcps/<id>/config.json` | 注入 Agent 原生配置（`config.toml` / `settings.json`） |
| **Plugins** | `~/.asm/store/plugins/<id>` | 符号链接 → Agent 插件目录 |

安装一次，即可为多个 Agent 启用，无需重复存储。

---

## 安装

```bash
git clone https://github.com/6xiaowu9/asm
cd asm
go build -o ~/.local/bin/asm .
```

需要 Go 1.22 及以上版本。

---

## 工作原理

```
来源（本地路径 / Git URL）
        │
        ▼
   ASM 存储区  (~/.asm/store/{skills,mcps,plugins}/<id>)
        │
        ├── 符号链接 ────────────► ~/.claude/skills/<id>
        ├── 符号链接 ────────────► ~/.codex/skills/<id>
        ├── 配置注入 ────────────► ~/.codex/config.toml
        └── 配置注入 ────────────► ~/.claude/settings.json
```

- **install** 将文件复制到存储区
- **enable / use** 在 Agent 目录中创建投影
- **migrate** 将已有的未托管包导入存储区
- **sync** 重建丢失的投影

---

## 配置

ASM 读取 `~/.asm/config.toml`，文件缺失时使用默认值。

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

环境变量 `ASM_HOME` 可在运行时覆盖 `asm_home`。

---

## 初始化

初始化 ASM，并将 `@ASM.md` 引用注入所有已配置 Agent 的配置文件。

```bash
asm init
```

`asm init` 是幂等的，可重复执行。

---

## Skills 命令

所有命令以 `asm skills` 为前缀。

### install

```bash
asm skills install <来源> [参数]
```

| 参数 | 说明 |
|------|------|
| `--id <string>` | 覆盖包 ID |
| `--subdir <路径>` | 指定来源中要安装的子目录 |
| `--ref <string>` | 指定 Git ref（分支、标签或 commit SHA） |
| `--agents <列表>` | 安装后立即为指定 Agent 启用（逗号分隔） |

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
asm skills status [--agent <名称>]
```

### enable / disable

```bash
asm skills enable <id> [--agent <名称>]
asm skills disable <id> [--agent <名称>]
```

### use

同时为多个 Agent 启用。

```bash
asm skills use <id> --agents claude,codex
```

### update

```bash
asm skills update <id>
asm skills update --all
```

### remove

移除所有投影、存储记录和文件。

```bash
asm skills remove <id>
```

### sync

重建存储中记录但磁盘上缺失的投影。

```bash
asm skills sync
```

### doctor

检查并报告损坏的投影。

```bash
asm skills doctor
```

### migrate

将 Agent 目录中未被 ASM 管理的 Skill 导入存储区。

```bash
asm skills migrate [--agent <名称>]
```

---

## MCP 命令

`asm mcp` 提供与 `asm skills` 相同的子命令集。MCP 包存储在 `~/.asm/store/mcps/<id>`，通过注入 Agent 原生配置文件（`~/.codex/config.toml` 或 `~/.claude/settings.json`）完成投影。

```bash
asm mcp install <来源> [参数]
asm mcp list
asm mcp status  [--agent <名称>]
asm mcp enable  <id> [--agent <名称>]
asm mcp disable <id> [--agent <名称>]
asm mcp use     <id> [--agents <列表>]
asm mcp update  <id> | --all
asm mcp remove  <id>
asm mcp sync
asm mcp doctor
asm mcp migrate [--agent <名称>]
```

`migrate` 会读取 Agent 原生 MCP 配置，将每个服务器导入存储区，将 `command` 路径改写为 ASM 存储副本，并删除 Agent 目录中原有的本地可执行文件。

---

## Plugins 命令

`asm plugins` 管理 Agent 原生插件扩展（如 superpowers）。插件存储在 `~/.asm/store/plugins/<id>`，并以符号链接方式投影到各 Agent 的插件目录。

```bash
asm plugins install <来源> [参数]
asm plugins list
asm plugins status  [--agent <名称>]
asm plugins enable  <id> [--agent <名称>]
asm plugins disable <id> [--agent <名称>]
asm plugins use     <id> [--agents <列表>]
asm plugins update  <id> | --all
asm plugins remove  <id>
asm plugins sync
asm plugins doctor
asm plugins migrate [--agent <名称>]
```

---

## 全局参数

| 参数 | 说明 |
|------|------|
| `--asm-home <路径>` | 覆盖 ASM 主目录 |
| `-h, --help` | 显示帮助信息 |

ASM 主目录解析优先级：`--asm-home` 参数 → `ASM_HOME` 环境变量 → `~/.asm`

---

## 目录结构

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
    └── <id> -> ~/.asm/store/skills/<id>    # 符号链接

~/.codex/
├── config.toml   # MCP 配置注入此处
└── skills/
    └── <id> -> ~/.asm/store/skills/<id>    # 符号链接
```
