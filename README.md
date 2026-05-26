# agent-notify

在 tmux（含 SSH 远程）中运行 Cursor CLI 与 Claude Code 时，通过 Agent hook 触发 Ghostty 桌面通知（OSC 777）。

**支持：** Cursor CLI（`cursor-agent`）、Claude Code  
**不支持：** Cursor IDE

## 前置条件

- [Ghostty](https://ghostty.org/) 终端（支持 OSC 777）
- tmux 3.2+（若在 tmux 内使用）

在每一层 tmux 的 `~/.tmux.conf` 中添加：

```tmux
set -g allow-passthrough on
```

## 安装

```bash
make install
# 或
go install ./cmd/agent-notify
```

## 配置 Hook

```bash
agent-notify install --all
agent-notify doctor
agent-notify test
```

配置文件：`~/.config/agent-notify/config.toml`

```toml
[events]
stop = true   # Agent 完成回复，等待输入
idle = false  # Claude 空闲 60s+（Notification hook）
tool = false  # shell/工具执行结束

[notify]
protocol = "osc777"
title_template = "{agent} — {context}"
body_stop = "等待输入"
```

## 命令

```bash
agent-notify send --event stop
agent-notify hook cursor stop    # Cursor CLI stop hook
agent-notify hook claude stop    # Claude Stop hook（输出 terminalSequence JSON）
agent-notify test
agent-notify doctor
agent-notify install --all [--force]
```

## Hook 配置位置

| Agent | 配置文件 | Hook 事件 |
|-------|---------|-----------|
| Cursor CLI | `~/.cursor/hooks.json` | `stop`, `afterShellExecution`（tool 开启时） |
| Claude Code | `~/.claude/settings.json` | `Stop`, `Notification`（idle 开启时） |

## 手动测试矩阵

```bash
# 1. 无 tmux（Ghostty 直接）
agent-notify test

# 2. 本地 tmux
tmux new-session -d 'agent-notify test'

# 3. 远程 tmux（SSH 到远程后在 tmux 内）
agent-notify test

# 4. 嵌套 tmux（本地 tmux → SSH → 远程 tmux）
# 确保两层 tmux 都设置了 allow-passthrough on
agent-notify test
```

## 工作原理

1. Agent hook 调用 `agent-notify hook ...`
2. CLI 生成 OSC 777 序列：`\033]777;notify;标题;正文\007`
3. 在 tmux 内优先写入 `client_tty`，否则用 DCS passthrough 透传
4. Claude Code 通过 hook JSON 的 `terminalSequence` 字段输出 OSC（hook 进程无 TTY）

## 开发

```bash
make test
make build
```
