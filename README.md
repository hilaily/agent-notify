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

# 两种测试路径（与 hook 发送方式一致）
agent-notify test cursor          # 始终会在 stderr 打印发送结果
agent-notify test cursor -v       # 同上（-v 保留兼容）
agent-notify test cursor --try-all  # 逐个尝试所有投递方式（调试用）
```

配置文件：`~/.config/agent-notify/config.toml`

```toml
[events]
stop = true   # Agent 完成回复，等待输入
idle = false  # Claude 空闲 60s+（Notification hook）
tool = false  # shell/工具执行结束

[notify]
protocol = "osc777"
title_template = "{agent} — {context}"  # context = 工作目录名
body_stop = "等待输入"

[inbox]
enabled = true
socket = "/run/user/1000/agent-notify.sock"
remote_socket = "/tmp/agent-notify-longbin.sock"
addr = "127.0.0.1:17777"
fallback_local = true
timeout_ms = 500
```

## 命令

```bash
agent-notify send --event stop
agent-notify hook cursor stop    # Cursor CLI stop hook
agent-notify hook claude stop    # Claude Stop hook（输出 terminalSequence JSON）
agent-notify inbox serve         # 本地接收远程通知记录
agent-notify inbox list          # 列出未处理通知
agent-notify inbox show <id>
agent-notify inbox done <id>
agent-notify inbox tui           # Bubble Tea TUI
agent-notify inbox ssh-config install  # 自动写 ~/.ssh/config RemoteForward
agent-notify test cursor [-v]
agent-notify test claude [--apply]
agent-notify doctor
agent-notify install --all [--force]
```

## 本地汇总 Inbox

在本地 Ghostty 所在机器启动接收服务：

```bash
agent-notify inbox serve
```

让命令自动写 SSH `RemoteForward` 配置：

```bash
agent-notify inbox ssh-config install
```

命令会在 `~/.ssh/config` 写入一个托管块，并把写入内容打印出来。已有 SSH 连接需要重连后才会生效。

默认写入的转发形式是：远程创建 `/tmp/agent-notify-$USER.sock`，转发到本地 `$XDG_RUNTIME_DIR/agent-notify.sock`。远程 hook 会尝试通过这个 SSH 反向转发把记录写回本地 inbox；如果本地接收服务不可用，会 fallback 写到远程机器自己的 `~/.local/state/agent-notify/inbox.jsonl`，避免丢记录。

查看和处理：

```bash
agent-notify inbox list
agent-notify inbox show <id>
agent-notify inbox done <id>
agent-notify inbox tui
```

## Hook 配置位置

| Agent | 配置文件 | Hook 事件 |
|-------|---------|-----------|
| Cursor CLI | `~/.cursor/hooks.json` | `stop`, `afterShellExecution`（tool 开启时） |
| Claude Code | `~/.claude/settings.json` | `Stop`, `Notification`（idle 开启时） |

## 手动测试矩阵

```bash
# 1. 无 tmux（Ghostty 直接）
agent-notify test cursor

# 2. 本地 tmux
tmux new-session -d 'agent-notify test cursor -v'

# 3. 远程 tmux（SSH 到远程后在 tmux 内）
agent-notify test cursor -v
# 应看到 delivery=passthrough-stdout

# 4. Claude 路径
agent-notify test claude --apply
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
make cross VERSION=v0.1.0   # 本地交叉编译
agent-notify version
```

## CI / Release

推送 `v*` tag 时，GitHub Actions 会构建并上传以下产物（PR 会跑测试和构建，但不发 Release）：

- `agent-notify-<version>_linux_amd64.tar.gz`
- `agent-notify-<version>_linux_arm64.tar.gz`
- `agent-notify-<version>_darwin_amd64.tar.gz`
- `agent-notify-<version>_darwin_arm64.tar.gz`

- **tag 发布**（如 `v0.1.0`）：版本号为 tag 名，并自动创建 GitHub Release

每个压缩包内含二进制和 `.sha256` 校验文件。
