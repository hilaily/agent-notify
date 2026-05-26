# agent-notify 设计规格

**日期：** 2026-05-26  
**状态：** 已批准（brainstorming）  
**目标：** 在本地/远程 tmux 中运行 Cursor CLI 与 Claude Code 时，通过 Agent hook 触发 OSC 777 桌面通知，经 tmux 透传至 Ghostty 终端。

---

## 1. 背景与目标

用户经常在本地 tmux 和远程 SSH tmux 中使用 AI Agent CLI（Cursor CLI、Claude Code）。当 Agent 完成一轮回复、回到等待输入状态时，用户希望收到桌面通知，而无需一直盯着终端。

Ghostty 终端支持 OSC 777（带标题/正文的桌面通知）和 OSC 9。tmux 默认会吞掉 OSC 序列，需通过 DCS passthrough 或写入 client TTY 透传。大部分 Agent CLI 支持 hook，可在 hook 中触发通知。

### 成功标准

- Agent 完成一轮回复、等待输入时，Ghostty 弹出桌面通知（默认开启）
- 通知标题包含 Agent 名称（Cursor / Claude）和项目目录或 tmux 窗口名
- 同一套 CLI 在以下场景均可工作：
  - 本地 Ghostty → 本地 tmux → Agent
  - 本地 Ghostty → SSH → 远程 tmux → Agent
  - 本地 Ghostty → 本地 tmux → SSH → 远程 tmux → Agent（嵌套 tmux）
- 提供 `agent-notify install` 一键写入 Cursor 与 Claude Code 的 hook 配置
- 触发事件可配置：stop（默认开）、idle（默认关）、tool（默认关）

### 非目标（首版）

- notify-send 等系统通知回退
- macOS / Windows 支持
- 不支持 Cursor IDE，仅支持 Cursor CLI（`cursor-agent`）
- Cursor CLI 的 `afterAgentResponse` hook（CLI 中不可靠）

---

## 2. 方案选择

在 brainstorming 中评估了三种方案：

| 方案 | 描述 | 结论 |
|------|------|------|
| A | 自研 `agent-notify` CLI + 安装脚本 | **选用** |
| B | 包装 soloterm/tnotify | 外部依赖，Claude terminalSequence 适配不内聚 |
| C | 纯 Shell 脚本 | 嵌套 tmux 逻辑难维护 |

---

## 3. 整体架构

```
┌─────────────┐     hook 触发      ┌──────────────────┐
│ Cursor CLI  │ ────────────────► │                  │
│ Claude Code │ ────────────────► │  agent-notify    │
└─────────────┘   stdin/env/flag   │  (核心 CLI)       │
                                    └────────┬─────────┘
                                             │ OSC 777
                                             ▼
                              ┌──────────────────────────┐
                              │ tmux 透传层 (0~N 层)      │
                              └────────────┬─────────────┘
                                             │ SSH (远程场景)
                                             ▼
                                    ┌─────────────┐
                                    │   Ghostty   │
                                    └─────────────┘
```

### 组件

1. **agent-notify CLI**（Go 单二进制）
   - `send`：发送通知（供 hook 或直接调用）
   - `hook`：Agent 专用入口，解析 stdin JSON
   - `install`：写入 hook 配置与默认 config
   - `test`：发送测试通知
   - `doctor`：检查 Ghostty/tmux/allow-passthrough 配置

2. **Hook 适配层**
   - Cursor：`~/.cursor/hooks.json`
   - Claude Code：`~/.claude/settings.json`

3. **配置文件**
   - `~/.config/agent-notify/config.toml`

---

## 4. 通知协议

### OSC 格式

Ghostty 优先使用 **OSC 777**（支持标题 + 正文）：

```
\033]777;notify;{title};{body}\007
```

OSC 9 作为备选（仅正文）：

```
\033]9;{body}\007
```

首版默认使用 OSC 777。

### tmux 透传策略

发送优先级：

1. **写 client TTY**（单层 tmux 最可靠）
   - `tmux display-message -p '#{client_tty}'`
   - 将 OSC 序列写入该 TTY

2. **DCS passthrough**（嵌套 tmux 必需）
   - 每层 tmux 包裹：`\033Ptmux;\033{inner}\033\\`
   - 嵌套 N 层则包裹 N 次

3. **直写 stdout**（无 tmux 且 hook 允许时）

### tmux 前置配置

用户需在涉及的每一层 tmux 中启用（安装脚本检测并提示）：

```tmux
set -g allow-passthrough on   # 需要 tmux 3.2+
```

### 远程 SSH 说明

- 远程 tmux 中写入 client TTY 时，数据经 SSH pty 传回本地
- 若本地还有 tmux，本地 tmux 也需 `allow-passthrough on`，否则 OSC 在本地被吞掉
- 嵌套 tmux（本地 tmux → SSH → 远程 tmux）需双层 passthrough 或双层 DCS 包裹

---

## 5. Hook 接入

### 触发事件映射

| 事件 | 含义 | Cursor CLI hook | Claude Code hook | 默认 |
|------|------|-----------------|------------------|------|
| stop | Agent 完成回复，等待输入 | `stop` | `Stop` | 开 |
| idle | 长时间无输入（约 60s） | 无等价 hook | `Notification` | 关 |
| tool | shell/工具执行结束 | `afterShellExecution` | `PostToolUse`（shell 类） | 关 |

### 通知内容

- **标题：** `{agent} — {context}`
  - `{agent}`：`Cursor` 或 `Claude`
  - `{context}`：tmux 窗口名（`#{window_name}`）；若无 tmux 则用 `basename(cwd)`
- **正文：** 按事件类型
  - stop：`等待输入`
  - idle：`空闲 60s+，等待输入`
  - tool：`工具执行完成`

模板可在 config.toml 中覆盖。

### Cursor CLI 集成

配置文件：`~/.cursor/hooks.json`（全局）或项目级 `.cursor/hooks.json`

```json
{
  "version": 1,
  "hooks": {
    "stop": [
      { "command": "agent-notify hook cursor stop" }
    ],
    "afterShellExecution": [
      { "command": "agent-notify hook cursor tool" }
    ]
  }
}
```

- `stop` hook 在 CLI 中可用
- hook 进程的 stdout 可能被捕获，CLI 内部通过写 TTY / DCS passthrough 发送 OSC，不依赖 stdout
- `afterShellExecution` 仅在 config 中 `events.tool = true` 时由 install 写入

### Claude Code 集成

配置文件：`~/.claude/settings.json`

Claude Code v2.1.139+ 的 hook 进程无 controlling TTY，**不可**直接写 `/dev/tty`。须通过 JSON 返回 `terminalSequence`，由 Claude Code 代为写入终端：

```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "agent-notify hook claude stop"
          }
        ]
      }
    ],
    "Notification": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "agent-notify hook claude idle"
          }
        ]
      }
    ]
  }
}
```

`agent-notify hook claude *` 输出：

```json
{"terminalSequence": "\033]777;notify;Cursor — myproject;等待输入\007"}
```

- `Stop` hook 必须检查 `stop_hook_active`：若为 true 则输出空 JSON 并 exit 0，避免无限循环
- `terminalSequence` 中的 OSC 序列由 Claude Code 写入其终端路径，天然兼容 tmux

---

## 6. CLI 接口

### 命令

```
agent-notify send [--title T] [--body B] [--event stop|idle|tool]
agent-notify hook cursor  stop|tool
agent-notify hook claude  stop|idle
agent-notify install [--cursor] [--claude] [--all] [--force]
agent-notify test
agent-notify doctor
```

### 环境变量（可选覆盖）

| 变量 | 含义 |
|------|------|
| `AGENT_NOTIFY_AGENT` | Agent 名称 |
| `AGENT_NOTIFY_CWD` | 工作目录 |
| `AGENT_NOTIFY_EVENT` | 事件类型 |

### 配置文件

路径：`~/.config/agent-notify/config.toml`

```toml
[events]
stop = true
idle = false
tool = false

[notify]
protocol = "osc777"   # osc777 | osc9
title_template = "{agent} — {context}"
body_stop = "等待输入"
body_idle = "空闲 60s+，等待输入"
body_tool = "工具执行完成"
```

`hook` 子命令读取 config，若对应 event 为 false 则静默 exit 0。

---

## 7. 安装流程

`agent-notify install --all` 执行：

1. 检测 `agent-notify` 是否在 PATH
2. 运行 `doctor`：检查是否在 tmux、tmux 版本、`allow-passthrough` 状态
3. 写入 `~/.config/agent-notify/config.toml`（不存在时）
4. 合并写入 Cursor `~/.cursor/hooks.json`（不覆盖已有同 event hook，除非 `--force`）
5. 合并写入 Claude `~/.claude/settings.json`
6. 运行 `agent-notify test` 验证通知

---

## 8. 错误处理

| 场景 | 行为 |
|------|------|
| 不在 tmux | 直接写 stdout（Cursor）或返回 terminalSequence（Claude） |
| tmux 无 client_tty | 回退 DCS passthrough |
| config 中 event 关闭 | hook 静默 exit 0 |
| Claude stop_hook_active=true | 不发送通知，输出 `{}` |
| doctor 发现 allow-passthrough 未开 | 打印修复提示，不阻断 install |
| OSC 发送失败 | exit 1，stderr 输出原因（hook 不应阻断 Agent） |

Cursor/Claude hook 脚本始终以 exit 0 结束（Claude Stop 除外需遵循 stop_hook_active 规则），避免影响 Agent 正常运行。

---

## 9. 技术选型

- **语言：** Go 1.22+
- **依赖：** 标准库为主；TOML 解析可用 `github.com/BurntSushi/toml`
- **分发：** `go install github.com/.../agent-notify@latest` 或仓库内 `make install`
- **平台：** Linux + Ghostty（首版）

---

## 10. 测试计划

### 单元测试

- OSC 777/9 序列生成
- tmux 层数检测与 DCS 多层包裹
- config 解析与 event 开关
- Claude hook JSON 输出格式

### 集成测试（手动）

| 场景 | 命令 | 期望 |
|------|------|------|
| 无 tmux | `agent-notify test` | Ghostty 弹出通知 |
| 本地 tmux | 在 tmux 内 `agent-notify test` | Ghostty 弹出通知 |
| 远程 tmux | SSH 到远程 tmux 内 test | 本地 Ghostty 弹出通知 |
| 嵌套 tmux | 本地 tmux → SSH → 远程 tmux test | 本地 Ghostty 弹出通知 |
| Cursor stop | cursor-agent 完成一轮 | 通知标题含 Cursor + 项目名 |
| Claude Stop | claude 完成一轮 | 通知标题含 Claude + 上下文 |

---

## 11. 项目结构（预期）

```
agent-notify/
├── cmd/agent-notify/main.go
├── internal/
│   ├── notify/       # OSC 生成与发送
│   ├── tmux/         # 层数检测、passthrough、client_tty
│   ├── hook/         # cursor/claude stdin 解析
│   ├── config/       # TOML 配置
│   └── install/      # hook 配置合并写入
├── docs/superpowers/specs/
│   └── 2026-05-26-agent-notify-design.md
├── go.mod
├── Makefile
└── README.md
```

---

## 12. 参考资料

- [Ghostty OSC 实现](https://github.com/ghostty-org/ghostty/blob/main/src/terminal/osc.zig)
- [Claude Code Hooks - terminalSequence](https://code.claude.com/docs/en/hooks)
- [Cursor Hooks 文档](https://cursor.com/docs/hooks)
- [tmux OSC passthrough（linw1995）](https://www.linw1995.com/en/agent-native-system-notifications/)
- [soloterm/tnotify](https://github.com/soloterm/tnotify)
