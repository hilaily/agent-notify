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
```

## 命令

```bash
agent-notify send --event stop
agent-notify hook cursor stop    # Cursor CLI stop hook
agent-notify hook claude stop    # Claude Stop hook（输出 terminalSequence JSON）
agent-notify test cursor [-v]
agent-notify test claude [--apply]
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

推送 `main` 分支或 `v*` tag 时，GitHub Actions 会构建并上传以下产物：

- `agent-notify-<version>_linux_amd64.tar.gz`
- `agent-notify-<version>_linux_arm64.tar.gz`
- `agent-notify-<version>_darwin_amd64.tar.gz`
- `agent-notify-<version>_darwin_arm64.tar.gz`

- **tag 发布**（如 `v0.1.0`）：版本号为 tag 名，并自动创建 GitHub Release
- **main 分支**：版本号为 `dev-<git-sha>`

每个压缩包内含二进制和 `.sha256` 校验文件。
