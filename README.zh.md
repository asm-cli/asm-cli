# ASM — AI Agent 技能包管理器

一个用于管理 AI Agent 技能包（Skill）和 MCP（Model Context Protocol）包的命令行工具，支持多 Agent 环境（Claude、Codex、Cursor、Gemini）。

[English](README.md)

---

## 目录

- [概述](#概述)
- [安装](#安装)
- [工作原理](#工作原理)
- [配置](#配置)
- [Skills 命令](#skills-命令)
- [MCP 命令](#mcp-命令)
- [全局参数](#全局参数)
- [目录结构](#目录结构)

---

## 概述

ASM 在 `~/.asm/` 下维护一个集中式包存储，并将包投影到各 Agent 的主目录中：

- **Skill 包**以符号链接方式投影（`~/.claude/skills/<id>` → `~/.asm/store/skills/<id>`）
- **MCP 包**以 JSON 清单文件方式投影（`~/.claude/mcp/<id>.json`）

安装一次，即可为多个 Agent 启用同一个包，无需重复存储。

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
   ASM 存储区  (~/.asm/store/skills/<id>  或  ~/.asm/store/mcps/<id>)
        │
        ├─── 符号链接 / JSON 清单 ──► ~/.claude/skills/<id>
        ├─── 符号链接 / JSON 清单 ──► ~/.codex/skills/<id>
        └─── ...
```

**install** 将文件复制到存储区。**enable / use** 在 Agent 目录中创建投影。**sync** 重建丢失的投影。

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

## Skills 命令

所有命令以 `asm skills` 为前缀。

### install

从本地目录或 Git URL 安装技能包。

```bash
asm skills install <来源> [参数]
```

| 参数 | 说明 |
|------|------|
| `--id <string>` | 覆盖包 ID（默认使用目录名或仓库名） |
| `--subdir <路径>` | 指定来源中要安装的子目录 |
| `--ref <string>` | 指定 Git ref（分支、标签或 commit SHA） |
| `--agents <列表>` | 安装完成后立即为指定 Agent 启用（逗号分隔） |

**示例：**

```bash
# 从本地目录安装
asm skills install ./my-skill

# 从 Git 安装并立即为 claude 启用
asm skills install https://github.com/user/my-skill --agents claude

# 从 Git 仓库的子目录安装，指定标签
asm skills install https://github.com/user/repo --subdir skills/my-skill --ref v1.2.0

# 使用自定义 ID 安装
asm skills install ./my-skill --id my-custom-id
```

### list

列出所有已安装的技能包。

```bash
asm skills list
```

### status

查看某个 Agent 的技能包启用/禁用状态。

```bash
asm skills status [--agent <名称>]
```

默认使用 `default_agents` 中的第一个 Agent（或 `claude`）。

### enable

为指定 Agent 创建投影并将技能包标记为已启用。

```bash
asm skills enable <id> [--agent <名称>]
```

### disable

为指定 Agent 移除投影并将技能包标记为已禁用。

```bash
asm skills disable <id> [--agent <名称>]
```

### use

同时为一个或多个 Agent 启用技能包。

```bash
asm skills use <id> [--agents <列表>]
```

```bash
asm skills use my-skill --agents claude,codex
```

### link

仅创建投影，不修改存储中的启用/禁用状态。

```bash
asm skills link <id> [--agent <名称>]
```

### unlink

仅移除投影，不修改存储中的启用/禁用状态。

```bash
asm skills unlink <id> [--agent <名称>]
```

### update

重新复制（本地）或重新拉取（Git）包文件以获取最新变更。

```bash
asm skills update <id>
asm skills update --all
```

### uninstall

移除存储记录和文件，但**不**移除现有投影（符号链接）。

```bash
asm skills uninstall <id>
```

### remove

移除所有投影、存储记录和已存储的文件。

```bash
asm skills remove <id>
```

### sync

重建存储中记录但磁盘上缺失的投影（例如迁移 Agent 主目录后）。

```bash
asm skills sync
```

### doctor

检查所有已跟踪的投影，报告损坏的条目。

```bash
asm skills doctor
```

### migrate

扫描 Agent 技能目录，找出尚未被 ASM 管理的条目。

```bash
asm skills migrate [--agent <名称>]
```

---

## MCP 命令

`asm mcp` 与 `asm skills` 提供相同的子命令集，区别在于包类型和投影格式：

- MCP 包存储在 `~/.asm/store/mcps/<id>`
- 投影为 JSON 清单文件，路径为 `~/.claude/mcp/<id>.json`

```bash
asm mcp install <来源> [参数]
asm mcp list
asm mcp status [--agent <名称>]
asm mcp enable <id> [--agent <名称>]
asm mcp disable <id> [--agent <名称>]
asm mcp use <id> [--agents <列表>]
asm mcp update <id> | --all
asm mcp remove <id>
asm mcp sync
asm mcp doctor
asm mcp migrate [--agent <名称>]
```

---

## 全局参数

| 参数 | 说明 |
|------|------|
| `--asm-home <路径>` | 为本次调用覆盖 ASM 主目录 |
| `-h, --help` | 显示任意命令的帮助信息 |

ASM 主目录的解析优先级：

1. `--asm-home` 参数
2. `ASM_HOME` 环境变量
3. `~/.asm`

---

## 目录结构

```
~/.asm/
├── config.toml
├── cache/
│   └── git/                  # Git 克隆缓存
└── store/
    ├── skills/
    │   ├── state.json        # 包和链接记录
    │   └── <id>/             # 已复制的技能文件
    └── mcps/
        ├── state.json
        └── <id>/             # 已复制的 MCP 文件

~/.claude/
├── skills/
│   └── <id> -> ~/.asm/store/skills/<id>   # 符号链接
└── mcp/
    └── <id>.json             # 生成的清单文件
```
