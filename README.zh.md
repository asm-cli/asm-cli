# ASM — AI Agent 包管理器

为 AI Agent 统一管理 Skills、MCP 服务器和插件。

[English](README.md)

---

## 解决的问题

同时使用多个 AI CLI 时，同一套 Skills、MCP 服务器和 Plugins 往往会被分别
安装到每个 CLI 自己的主目录中。这会带来重复存储、版本不一致、重复配置、
以及不同 Agent 之间能力漂移的问题。

ASM 通过在 `~/.asm/` 中集中存储包，并按需投影到 Claude、Codex、Gemini、
Cursor 等 Agent 环境，解决多 CLI 重复存储同一套 Skills、MCP 和 Plugins
的问题。

---

## 目录

- [解决的问题](#解决的问题)
- [概述](#概述)
- [安装](#安装)
- [快速开始](#快速开始)
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

使用 `curl` 安装最新 release 二进制：

```bash
mkdir -p ~/.local/bin
tmp="$(mktemp -d)"
curl -fsSL "https://github.com/asm-cli/asm-cli/releases/latest/download/asm-latest-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/amd64/').tar.gz" \
  | tar -xz -C "$tmp"
install -m 0755 "$tmp"/*/asm ~/.local/bin/asm
rm -rf "$tmp"
```

该 `curl` 安装命令匹配 Linux amd64 和 macOS arm64 的 release 资产。Windows
用户可以从 GitHub Release 页面下载 `asm-latest-windows-amd64.zip`，并将
`asm.exe` 放到 `PATH` 中的目录。

如果 `~/.local/bin` 还不在 shell 的 `PATH` 中，请添加：

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

验证安装：

```bash
asm version
```

也可以从源码构建：

```bash
git clone https://github.com/asm-cli/asm-cli
cd asm-cli
go build -o ~/.local/bin/asm .
```

需要 Go 1.22 及以上版本。

---

## 快速开始

先初始化 ASM。该命令会创建 `~/.asm/`，写入默认配置，并将 `@ASM.md`
引用注入已检测到的 Agent 配置。

```bash
asm init
```

### 在 shell 中使用

Skill、MCP 服务器或 Plugin 只需安装一次，然后为一个或多个 Agent 启用。

```bash
asm skills install https://github.com/user/my-skill --agents claude,codex
asm mcp install https://github.com/user/my-mcp --agents codex
asm plugins install https://github.com/user/my-plugin --agents claude,codex
```

查看 ASM 管理的内容，并检查投影是否健康。

```bash
asm skills list
asm skills status --agent codex
asm skills doctor
```

如果包已经存在于 ASM 存储区，请直接 enable 或 use，不要重复安装。

```bash
asm skills enable my-skill --agent claude
asm skills use my-skill --agents claude,codex
```

### 在 AI CLI 中使用

执行 `asm init` 后，受支持的 AI CLI 会收到 `@ASM.md` 指令。你可以让
Agent 直接使用 ASM 包管理命令，也可以在 CLI 支持时使用 slash command
映射。

```text
/skills install https://github.com/user/my-skill --agents claude,codex
/mcp install https://github.com/user/my-mcp --agents codex
/plugins install https://github.com/user/my-plugin --agents claude,codex
```

这些命令会映射到相同的 ASM 命令：

```text
/skills   -> asm skills
/mcp      -> asm mcp
/plugins  -> asm plugins
```

例如，在 AI CLI 的对话中输入：

```text
Use ASM to install https://github.com/user/my-skill for claude and codex.
```

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
