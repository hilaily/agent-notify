# Agent Notify Inbox Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Collect agent notifications from local and remote machines into a local pending list that can be viewed first via CLI and later via a Bubble Tea TUI.

**Architecture:** Add an `internal/inbox` package for records, JSONL storage, local receiver server, remote uploader, and SSH config setup. Hooks will keep sending Ghostty notifications, then attempt to upload an inbox record to the configured local receiver; if upload fails, they write a local fallback JSONL record. CLI commands operate on the local JSONL store, while `inbox serve` receives remote records over Unix socket or TCP through SSH `RemoteForward`.

**Tech Stack:** Go stdlib (`net/http`, `net`, JSONL files, `flag`), existing `config`, `context`, `hook`, `logx`, `tmux`; Bubble Tea added only when implementing the TUI phase.

---

### Task 1: Config Model For Inbox

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

**Step 1: Add failing config tests**

Add tests that assert:
- `config.Default().Inbox.Enabled == true`
- default socket path is non-empty and prefers `$XDG_RUNTIME_DIR/agent-notify.sock` when available
- default TCP address is `127.0.0.1:17777`
- TOML can override `[inbox] enabled`, `socket`, `addr`, `fallback_local`, and `timeout_ms`

**Step 2: Run the focused test**

Run: `go test ./internal/config`

Expected: FAIL because `Inbox` config does not exist yet.

**Step 3: Implement inbox config**

Add:

```go
type Inbox struct {
	Enabled       bool   `toml:"enabled"`
	Socket        string `toml:"socket"`
	Addr          string `toml:"addr"`
	FallbackLocal bool   `toml:"fallback_local"`
	TimeoutMS     int    `toml:"timeout_ms"`
}
```

Update `Config` and `Default()`:
- `Enabled: true`
- `Socket: DefaultInboxSocket()`
- `Addr: "127.0.0.1:17777"`
- `FallbackLocal: true`
- `TimeoutMS: 500`

Add `DefaultInboxSocket()` in `config`:
- if `$XDG_RUNTIME_DIR` is set, return `$XDG_RUNTIME_DIR/agent-notify.sock`
- otherwise return `$HOME/.local/state/agent-notify/agent-notify.sock`

**Step 4: Run focused test again**

Run: `go test ./internal/config`

Expected: PASS.

---

### Task 2: Inbox Record And JSONL Store

**Files:**
- Create: `internal/inbox/record.go`
- Create: `internal/inbox/store.go`
- Create: `internal/inbox/store_test.go`

**Step 1: Add failing store tests**

Cover:
- appending records creates `~/.local/state/agent-notify/inbox.jsonl`
- `List` returns records in file order
- `Pending` filters `status == "pending"`
- `MarkDone(ids...)` rewrites only matching pending records to `done`
- invalid JSONL lines are skipped, not fatal

**Step 2: Run focused test**

Run: `go test ./internal/inbox`

Expected: FAIL because package does not exist yet.

**Step 3: Implement record model**

Record fields:
- `id`
- `time`
- `host`
- `agent`
- `event`
- `cwd`
- `title`
- `body`
- `status`
- `source`
- `tmux`

Use a nested `TmuxContext` with `session`, `window`, `pane`.

Generate IDs with timestamp + random suffix from stdlib, for example `20260526-165001-a1b2c3`.

**Step 4: Implement JSONL store**

Store path:
- default `~/.local/state/agent-notify/inbox.jsonl`
- injectable path for tests

Operations:
- `Append(record Record) error`
- `List() ([]Record, error)`
- `Pending() ([]Record, error)`
- `MarkDone(ids []string) (int, error)`
- `Remove(ids []string) (int, error)`
- `ClearDone() (int, error)`
- `ClearAll() (int, error)`

Rewrite operations should write to a temp file and rename.

**Step 5: Run tests**

Run: `go test ./internal/inbox`

Expected: PASS.

---

### Task 3: Inbox HTTP Receiver And Uploader

**Files:**
- Create: `internal/inbox/server.go`
- Create: `internal/inbox/client.go`
- Create: `internal/inbox/server_test.go`

**Step 1: Add failing server/client tests**

Cover:
- `POST /inbox` appends a valid record
- invalid method returns 405
- invalid JSON returns 400
- client can POST to an HTTP test server
- client timeout is respected by using a small timeout

**Step 2: Run focused test**

Run: `go test ./internal/inbox`

Expected: FAIL for missing server/client.

**Step 3: Implement receiver**

Implement an HTTP handler:
- `POST /inbox`
- decode `Record`
- fill missing `id`, `time`, and `status=pending`
- append to store
- return JSON `{ "ok": true, "id": "..." }`

**Step 4: Implement uploader**

Uploader behavior:
- prefer Unix socket when configured
- support TCP address fallback through `http://127.0.0.1:17777/inbox`
- timeout from config
- no retries in hook path

For Unix socket, use an `http.Client` with custom `Transport.DialContext`.

**Step 5: Run focused tests**

Run: `go test ./internal/inbox`

Expected: PASS.

---

### Task 4: Build Records From Hook Context

**Files:**
- Create: `internal/inbox/build.go`
- Create: `internal/inbox/build_test.go`
- Modify: `internal/tmux` only if existing APIs do not expose session/window/pane cleanly

**Step 1: Add failing build tests**

Cover:
- host is populated from `os.Hostname`
- cwd/title/body/agent/event are copied
- tmux fields are empty outside tmux
- source can be `local` or `remote`

**Step 2: Run focused test**

Run: `go test ./internal/inbox`

Expected: FAIL.

**Step 3: Implement builder**

Add a builder function that takes agent, event, cwd, title, body and returns a complete pending `Record`.

If tmux env vars are enough, capture:
- `TMUX_PANE` as pane
- session/window best effort from `tmux display-message` if current tmux helpers already support shelling out

Do not block hook completion if tmux metadata cannot be collected.

**Step 4: Run test**

Run: `go test ./internal/inbox`

Expected: PASS.

---

### Task 5: Hook Integration With Upload And Local Fallback

**Files:**
- Modify: `internal/hook/cursor.go`
- Modify: `internal/hook/claude.go`
- Modify: `internal/hook/hook_test.go`

**Step 1: Add failing hook tests**

Cover:
- enabled cursor hook calls the inbox uploader once after notification path
- disabled hook does not record
- upload failure with `FallbackLocal=true` appends local fallback
- upload failure with `FallbackLocal=false` does not append fallback
- Claude hook records after writing `terminalSequence`

Use package-level function variables for uploader/store append just like existing `sendForHook` stubs.

**Step 2: Run focused tests**

Run: `go test ./internal/hook`

Expected: FAIL.

**Step 3: Implement integration**

After notification is successfully generated/sent:
- build inbox record
- if `cfg.Inbox.Enabled`, try upload
- on upload error, log `inbox upload failed`
- if fallback enabled, append local record with `source="fallback"`

Avoid returning inbox upload/fallback errors from hook unless local fallback write itself is unexpectedly fatal and needs surfacing. Notification behavior should remain primary.

**Step 4: Run focused tests**

Run: `go test ./internal/hook`

Expected: PASS.

---

### Task 6: CLI Inbox Commands

**Files:**
- Modify: `cmd/agent-notify/main.go`
- Create: `cmd/agent-notify/inbox.go`
- Create: `cmd/agent-notify/inbox_test.go`

**Step 1: Add failing command tests**

Cover:
- `inbox list` prints pending records
- `inbox list --all` includes done records
- `inbox show <id>` prints details
- `inbox done <id...>` marks records done and prints count
- `inbox rm <id...>` removes records
- `inbox clear --done` clears done records

**Step 2: Run command tests**

Run: `go test ./cmd/agent-notify`

Expected: FAIL.

**Step 3: Implement command routing**

Add `inbox` to top-level `run`.

Subcommands:
- `agent-notify inbox list [--all|--done|--pending]`
- `agent-notify inbox show <id>`
- `agent-notify inbox done <id...>`
- `agent-notify inbox rm <id...>`
- `agent-notify inbox clear [--done]`

Keep output simple and scriptable:
- one line per list item
- short ID, local time, status, host, agent/event, cwd, title

**Step 4: Run focused tests**

Run: `go test ./cmd/agent-notify`

Expected: PASS.

---

### Task 7: Inbox Serve Command

**Files:**
- Modify: `cmd/agent-notify/inbox.go`
- Create or modify: `cmd/agent-notify/inbox_test.go`

**Step 1: Add failing tests**

Cover parsing and listener selection:
- `serve --socket <path>` uses Unix socket mode
- `serve --addr 127.0.0.1:17777` uses TCP mode
- missing flags default to config socket

Do not write an infinite server test through `main`; test smaller functions.

**Step 2: Run focused test**

Run: `go test ./cmd/agent-notify`

Expected: FAIL.

**Step 3: Implement serve**

Add:
- `agent-notify inbox serve [--socket PATH] [--addr ADDR]`

Behavior:
- default to Unix socket from config
- remove stale socket before listen
- chmod socket `0600`
- print listening location to stderr
- serve until interrupted

**Step 4: Run focused tests**

Run: `go test ./cmd/agent-notify`

Expected: PASS.

---

### Task 8: SSH Config Auto Writer

**Files:**
- Create: `internal/inbox/sshconfig.go`
- Create: `internal/inbox/sshconfig_test.go`
- Modify: `cmd/agent-notify/inbox.go`
- Modify: `cmd/agent-notify/inbox_test.go`

**Step 1: Add failing SSH config tests**

Cover:
- empty file gets a managed block
- existing managed block is replaced, not duplicated
- unrelated user config is preserved
- generated block uses the configured socket path
- command returns the exact block so CLI can print it after writing

**Step 2: Run focused tests**

Run: `go test ./internal/inbox ./cmd/agent-notify`

Expected: FAIL.

**Step 3: Implement managed block**

Use markers:

```text
# BEGIN agent-notify inbox
Host *
    RemoteForward <socket> <socket>
    ExitOnForwardFailure no
    ServerAliveInterval 30
# END agent-notify inbox
```

Write to `~/.ssh/config` by default.

Rules:
- create `~/.ssh` as `0700`
- create config as `0600`
- preserve existing content
- replace old managed block idempotently
- do not parse or modify other `Host` sections

**Step 4: Add CLI command**

Add:
- `agent-notify inbox ssh-config install [--path PATH] [--socket PATH]`

After writing, print:
- target file path
- the exact managed block that was written
- one reminder that existing SSH sessions must reconnect for `RemoteForward` to take effect

**Step 5: Run focused tests**

Run: `go test ./internal/inbox ./cmd/agent-notify`

Expected: PASS.

---

### Task 9: Documentation For CLI Inbox

**Files:**
- Modify: `README.md`

**Step 1: Document local receiver setup**

Add:

```bash
agent-notify inbox serve
```

**Step 2: Document SSH config auto setup**

Add:

```bash
agent-notify inbox ssh-config install
```

Explain that the command writes a managed `Host *` block and prints it after writing.

**Step 3: Document list workflow**

Add:

```bash
agent-notify inbox list
agent-notify inbox show <id>
agent-notify inbox done <id>
```

**Step 4: Run docs-adjacent checks**

Run: `go test ./...`

Expected: PASS.

---

### Task 10: Bubble Tea TUI

**Files:**
- Modify: `go.mod`
- Create: `internal/inbox/tui.go`
- Modify: `cmd/agent-notify/inbox.go`

**Step 1: Add dependency**

Run:

```bash
go get github.com/charmbracelet/bubbletea
```

**Step 2: Implement initial TUI**

Add:
- `agent-notify inbox tui`

Features:
- list pending items
- `j/k` or arrows move
- `enter` opens details in-place
- `d` marks selected item done
- `r` reloads file
- `q` quits

No Ghostty/tmux jump behavior in this phase.

**Step 3: Run full verification**

Run:

```bash
go test ./...
make build
```

Expected: PASS.

---

### Task 11: Final Verification

**Files:**
- All touched files

**Step 1: Run full test suite**

Run:

```bash
go test ./...
```

Expected: PASS.

**Step 2: Build binary**

Run:

```bash
make build
```

Expected: PASS.

**Step 3: Manual smoke test**

Run:

```bash
tmpdir="$(mktemp -d)"
XDG_RUNTIME_DIR="$tmpdir" HOME="$tmpdir/home" ./bin/agent-notify inbox ssh-config install
XDG_RUNTIME_DIR="$tmpdir" HOME="$tmpdir/home" ./bin/agent-notify inbox serve --socket "$tmpdir/agent-notify.sock"
```

In another shell, send a test POST or run a hook simulation and confirm:

```bash
XDG_RUNTIME_DIR="$tmpdir" HOME="$tmpdir/home" ./bin/agent-notify inbox list
```

Expected: the test record appears as pending.
