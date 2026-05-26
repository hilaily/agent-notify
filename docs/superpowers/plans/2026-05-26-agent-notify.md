# agent-notify Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go CLI that sends Ghostty desktop notifications via OSC 777 through tmux passthrough, with installable hooks for Cursor CLI and Claude Code.

**Architecture:** Single Go binary with focused internal packages: `config` (TOML), `notify` (OSC encode/send), `tmux` (layer detection + passthrough), `context` (title/body templates), `hook` (Cursor/Claude stdin adapters), `install` (merge hook JSON). CLI subcommands wired in `cmd/agent-notify/main.go`.

**Tech Stack:** Go 1.22+, `github.com/BurntSushi/toml`, standard library only otherwise.

**Spec:** `docs/superpowers/specs/2026-05-26-agent-notify-design.md`

---

## File Structure

| File | Responsibility |
|------|----------------|
| `go.mod` | Module definition |
| `cmd/agent-notify/main.go` | CLI entry, subcommand dispatch |
| `internal/config/config.go` | Load/merge TOML defaults |
| `internal/config/config_test.go` | Config parsing tests |
| `internal/notify/osc.go` | Build OSC 777/9 byte sequences |
| `internal/notify/osc_test.go` | OSC sequence tests |
| `internal/notify/send.go` | Write OSC to stdout/client_tty/DCS |
| `internal/notify/send_test.go` | Send logic tests (mock writer) |
| `internal/tmux/tmux.go` | Detect tmux, client_tty, passthrough wrap |
| `internal/tmux/tmux_test.go` | Passthrough wrapping tests |
| `internal/context/meta.go` | Resolve agent/cwd/window, render templates |
| `internal/context/meta_test.go` | Template rendering tests |
| `internal/hook/cursor.go` | Parse Cursor hook stdin, call send |
| `internal/hook/claude.go` | Parse Claude hook stdin, emit terminalSequence JSON |
| `internal/hook/hook_test.go` | Hook handler tests |
| `internal/install/install.go` | Write config + merge hooks.json/settings.json |
| `internal/install/install_test.go` | Merge logic tests |
| `Makefile` | build, test, install targets |
| `README.md` | Usage, tmux setup, hook install |

---

### Task 1: Project Bootstrap

**Files:**
- Create: `go.mod`
- Create: `cmd/agent-notify/main.go`
- Create: `Makefile`

- [ ] **Step 1: Initialize Go module**

```bash
cd /home/longbin/code/agent-notify
go mod init github.com/longbin/agent-notify
```

- [ ] **Step 2: Create minimal main with subcommand stub**

Create `cmd/agent-notify/main.go`:

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "send", "hook", "install", "test", "doctor", "help", "-h", "--help":
		fmt.Fprintf(os.Stderr, "agent-notify: %s not implemented yet\n", os.Args[1])
		os.Exit(1)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprint(os.Stderr, `Usage: agent-notify <command>
Commands:
  send     Send a notification
  hook     Agent hook entrypoint
  install  Install hook configs
  test     Send test notification
  doctor   Check environment
`)
}
```

- [ ] **Step 3: Create Makefile**

```makefile
.PHONY: build test install
BINARY := agent-notify

build:
	go build -o bin/$(BINARY) ./cmd/agent-notify

test:
	go test ./...

install: build
	install -m 755 bin/$(BINARY) $(HOME)/.local/bin/$(BINARY)
```

- [ ] **Step 4: Verify build**

Run: `go build -o bin/agent-notify ./cmd/agent-notify`
Expected: succeeds, no output

- [ ] **Step 5: Commit**

```bash
git init
git add go.mod cmd/agent-notify/main.go Makefile
git commit -m "chore: bootstrap agent-notify Go project"
```

---

### Task 2: Config Package

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/config/config_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	if !cfg.Events.Stop {
		t.Fatal("expected stop=true by default")
	}
	if cfg.Events.Idle {
		t.Fatal("expected idle=false by default")
	}
	if cfg.Notify.Protocol != "osc777" {
		t.Fatalf("expected osc777, got %q", cfg.Notify.Protocol)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[events]
stop = false
tool = true

[notify]
body_stop = "custom stop"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Events.Stop {
		t.Fatal("expected stop=false")
	}
	if !cfg.Events.Tool {
		t.Fatal("expected tool=true")
	}
	if cfg.Notify.BodyStop != "custom stop" {
		t.Fatalf("got %q", cfg.Notify.BodyStop)
	}
}

func TestEventEnabled(t *testing.T) {
	cfg := Default()
	cfg.Events.Stop = false
	if cfg.EventEnabled("stop") {
		t.Fatal("expected stop disabled")
	}
	if !cfg.EventEnabled("STOP") {
		t.Fatal("expected case-insensitive match")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/... -v`
Expected: FAIL — package/config not defined

- [ ] **Step 3: Implement config**

```bash
go get github.com/BurntSushi/toml
```

Create `internal/config/config.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Events Events `toml:"events"`
	Notify Notify `toml:"notify"`
}

type Events struct {
	Stop bool `toml:"stop"`
	Idle bool `toml:"idle"`
	Tool bool `toml:"tool"`
}

type Notify struct {
	Protocol      string `toml:"protocol"`
	TitleTemplate string `toml:"title_template"`
	BodyStop      string `toml:"body_stop"`
	BodyIdle      string `toml:"body_idle"`
	BodyTool      string `toml:"body_tool"`
}

func Default() Config {
	return Config{
		Events: Events{Stop: true, Idle: false, Tool: false},
		Notify: Notify{
			Protocol:      "osc777",
			TitleTemplate: "{agent} — {context}",
			BodyStop:      "等待输入",
			BodyIdle:      "空闲 60s+，等待输入",
			BodyTool:      "工具执行完成",
		},
	}
}

func DefaultPath() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "agent-notify", "config.toml")
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func LoadDefault() (Config, error) {
	return Load(DefaultPath())
}

func (c Config) EventEnabled(event string) bool {
	switch strings.ToLower(event) {
	case "stop":
		return c.Events.Stop
	case "idle":
		return c.Events.Idle
	case "tool":
		return c.Events.Tool
	default:
		return false
	}
}

func (c Config) BodyForEvent(event string) string {
	switch strings.ToLower(event) {
	case "idle":
		return c.Notify.BodyIdle
	case "tool":
		return c.Notify.BodyTool
	default:
		return c.Notify.BodyStop
	}
}

func (c Config) WriteDefault(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(Default())
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/config/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config go.mod go.sum
git commit -m "feat: add TOML config with defaults and event toggles"
```

---

### Task 3: OSC Sequence Builder

**Files:**
- Create: `internal/notify/osc.go`
- Create: `internal/notify/osc_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/notify/osc_test.go`:

```go
package notify

import "testing"

func TestBuildOSC777(t *testing.T) {
	seq := BuildSequence("osc777", "Claude — proj", "等待输入")
	want := "\033]777;notify;Claude — proj;等待输入\007"
	if seq != want {
		t.Fatalf("got %q want %q", seq, want)
	}
}

func TestBuildOSC9(t *testing.T) {
	seq := BuildSequence("osc9", "ignored", "等待输入")
	want := "\033]9;等待输入\007"
	if seq != want {
		t.Fatalf("got %q want %q", seq, want)
	}
}

func TestSemicolonInTitleEscaped(t *testing.T) {
	seq := BuildSequence("osc777", "a;b", "body")
	if seq != "\033]777;notify;a\\;b;body\007" {
		t.Fatalf("unexpected escape: %q", seq)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/notify/... -run TestBuild -v`
Expected: FAIL

- [ ] **Step 3: Implement OSC builder**

Create `internal/notify/osc.go`:

```go
package notify

import "strings"

func BuildSequence(protocol, title, body string) string {
	switch protocol {
	case "osc9":
		return "\033]9;" + escapeOSCField(body) + "\007"
	default:
		return "\033]777;notify;" + escapeOSCField(title) + ";" + escapeOSCField(body) + "\007"
	}
}

func escapeOSCField(s string) string {
	return strings.ReplaceAll(s, ";", "\\;")
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/notify/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/notify/osc.go internal/notify/osc_test.go
git commit -m "feat: add OSC 777/9 sequence builder"
```

---

### Task 4: tmux Passthrough Layer

**Files:**
- Create: `internal/tmux/tmux.go`
- Create: `internal/tmux/tmux_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/tmux/tmux_test.go`:

```go
package tmux

import "testing"

func TestInTmux(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-123,1,0")
	if !InTmux() {
		t.Fatal("expected InTmux true")
	}
}

func TestWrapPassthroughSingle(t *testing.T) {
	inner := "\033]777;notify;t;b\007"
	got := WrapPassthrough(inner)
	want := "\033Ptmux;\033" + inner + "\033\\"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestWrapPassthroughNested(t *testing.T) {
	inner := "\033]777;notify;t;b\007"
	got := WrapPassthroughLayers(inner, 2)
	once := WrapPassthrough(inner)
	twice := WrapPassthrough(once)
	if got != twice {
		t.Fatalf("nested wrap mismatch")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tmux/... -v`
Expected: FAIL

- [ ] **Step 3: Implement tmux helpers**

Create `internal/tmux/tmux.go`:

```go
package tmux

import (
	"os"
	"os/exec"
	"strings"
)

func InTmux() bool {
	return os.Getenv("TMUX") != ""
}

func TmuxLayerCount() int {
	if !InTmux() {
		return 0
	}
	// Each TMUX env var in nested attach typically appears once per server attach chain.
	// Conservative: count commas segments; fallback 1 if in tmux.
	parts := strings.Split(os.Getenv("TMUX"), ",")
	if len(parts) >= 1 {
		return 1
	}
	return 0
}

func ClientTTY() (string, error) {
	if !InTmux() {
		return "", nil
	}
	out, err := exec.Command("tmux", "display-message", "-p", "#{client_tty}").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func WrapPassthrough(seq string) string {
	return "\033Ptmux;\033" + seq + "\033\\"
}

func WrapPassthroughLayers(seq string, layers int) string {
	out := seq
	for i := 0; i < layers; i++ {
		out = WrapPassthrough(out)
	}
	return out
}

func AllowPassthroughEnabled() (bool, string, error) {
	if !InTmux() {
		return true, "", nil
	}
	out, err := exec.Command("tmux", "show-option", "-gv", "allow-passthrough").Output()
	if err != nil {
		return false, "", err
	}
	val := strings.TrimSpace(string(out))
	return val == "on" || val == "all", val, nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/tmux/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tmux
git commit -m "feat: add tmux passthrough and client_tty helpers"
```

---

### Task 5: Notification Sender

**Files:**
- Create: `internal/notify/send.go`
- Create: `internal/notify/send_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/notify/send_test.go`:

```go
package notify

import (
	"bytes"
	"testing"
)

func TestSendDirect(t *testing.T) {
	var buf bytes.Buffer
	err := Send(SendOptions{
		Protocol: "osc777",
		Title:    "Cursor — app",
		Body:     "等待输入",
		Writer:   &buf,
		InTmux:   false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("777;notify")) {
		t.Fatalf("missing osc777: %q", buf.String())
	}
}

func TestSendTmuxUsesPassthroughWhenNoTTY(t *testing.T) {
	var buf bytes.Buffer
	err := Send(SendOptions{
		Protocol: "osc777",
		Title:    "t",
		Body:     "b",
		Writer:   &buf,
		InTmux:   true,
		Layers:   1,
		ClientTTY: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("\033Ptmux;")) {
		t.Fatalf("expected passthrough prefix, got %q", buf.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/notify/... -run TestSend -v`
Expected: FAIL

- [ ] **Step 3: Implement sender**

Create `internal/notify/send.go`:

```go
package notify

import (
	"fmt"
	"io"
	"os"

	"github.com/longbin/agent-notify/internal/tmux"
)

type SendOptions struct {
	Protocol  string
	Title     string
	Body      string
	Writer    io.Writer
	InTmux    bool
	Layers    int
	ClientTTY string
}

func Send(opts SendOptions) error {
	seq := BuildSequence(opts.Protocol, opts.Title, opts.Body)
	w := opts.Writer
	if w == nil {
		w = os.Stdout
	}

	if opts.InTmux && opts.ClientTTY != "" {
		f, err := os.OpenFile(opts.ClientTTY, os.O_WRONLY, 0)
		if err == nil {
			defer f.Close()
			_, err = io.WriteString(f, seq)
			return err
		}
	}

	out := seq
	if opts.InTmux {
		layers := opts.Layers
		if layers <= 0 {
			layers = 1
		}
		out = tmux.WrapPassthroughLayers(seq, layers)
	}
	_, err := io.WriteString(w, out)
	return err
}

func SendAuto(protocol, title, body string) error {
	inTmux := tmux.InTmux()
	clientTTY, _ := tmux.ClientTTY()
	return Send(SendOptions{
		Protocol:  protocol,
		Title:     title,
		Body:      body,
		InTmux:    inTmux,
		Layers:    tmuxLayerCountSafe(),
		ClientTTY: clientTTY,
	})
}

func tmuxLayerCountSafe() int {
	if !tmux.InTmux() {
		return 0
	}
	return 1
}

func TestNotification(title, body string) error {
	if err := SendAuto("osc777", title, body); err != nil {
		return fmt.Errorf("send test notification: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/notify/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/notify/send.go internal/notify/send_test.go
git commit -m "feat: send OSC via stdout, client_tty, or tmux passthrough"
```

---

### Task 6: Context / Template Rendering

**Files:**
- Create: `internal/context/meta.go`
- Create: `internal/context/meta_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/context/meta_test.go`:

```go
package context

import "testing"

func TestRenderTitle(t *testing.T) {
	meta := Meta{Agent: "Cursor", Context: "myapp"}
	got := Render("{agent} — {context}", meta)
	want := "Cursor — myapp"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestContextFromCWD(t *testing.T) {
	meta := Meta{Agent: "Claude", CWD: "/home/user/code/myapp"}
	if meta.ResolveContext("") != "myapp" {
		t.Fatalf("got %q", meta.ResolveContext(""))
	}
}

func TestContextPrefersWindow(t *testing.T) {
	meta := Meta{Agent: "Claude", CWD: "/home/user/code/myapp"}
	if meta.ResolveContext("tmux-win") != "tmux-win" {
		t.Fatalf("expected window name")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/context/... -v`
Expected: FAIL

- [ ] **Step 3: Implement meta**

Create `internal/context/meta.go`:

```go
package context

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Meta struct {
	Agent   string
	CWD     string
	Context string
	Event   string
}

func Render(tmpl string, m Meta) string {
	ctx := m.ResolveContext(m.Context)
	out := strings.ReplaceAll(tmpl, "{agent}", m.Agent)
	out = strings.ReplaceAll(out, "{context}", ctx)
	return out
}

func (m Meta) ResolveContext(window string) string {
	if window != "" {
		return window
	}
	if m.Context != "" {
		return m.Context
	}
	cwd := m.CWD
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	if cwd == "" {
		return "unknown"
	}
	return filepath.Base(cwd)
}

func TmuxWindowName() string {
	if os.Getenv("TMUX") == "" {
		return ""
	}
	out, err := exec.Command("tmux", "display-message", "-p", "#{window_name}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func MetaFromEnv(agent, event string) Meta {
	cwd := os.Getenv("AGENT_NOTIFY_CWD")
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	if a := os.Getenv("AGENT_NOTIFY_AGENT"); a != "" {
		agent = a
	}
	if e := os.Getenv("AGENT_NOTIFY_EVENT"); e != "" {
		event = e
	}
	return Meta{
		Agent:   agent,
		CWD:     cwd,
		Context: TmuxWindowName(),
		Event:   event,
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/context/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/context
git commit -m "feat: resolve notification title context from tmux or cwd"
```

---

### Task 7: Hook Handlers (Cursor + Claude)

**Files:**
- Create: `internal/hook/cursor.go`
- Create: `internal/hook/claude.go`
- Create: `internal/hook/hook_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/hook/hook_test.go`:

```go
package hook

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/longbin/agent-notify/internal/config"
)

func TestCursorStopHookDisabled(t *testing.T) {
	cfg := config.Default()
	cfg.Events.Stop = false
	err := RunCursor(bytes.NewReader([]byte(`{"workspace_roots":["/tmp/proj"]}`)), cfg, "stop", &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestClaudeStopOutputsTerminalSequence(t *testing.T) {
	cfg := config.Default()
	var out bytes.Buffer
	err := RunClaude(strings.NewReader(`{"stop_hook_active":false}`), cfg, "stop", &out)
	if err != nil {
		t.Fatal(err)
	}
	var resp map[string]string
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp["terminalSequence"], "777;notify") {
		t.Fatalf("bad sequence: %v", resp)
	}
}

func TestClaudeStopHookActiveSkips(t *testing.T) {
	cfg := config.Default()
	var out bytes.Buffer
	err := RunClaude(strings.NewReader(`{"stop_hook_active":true}`), cfg, "stop", &out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "{}" {
		t.Fatalf("expected {}, got %q", out.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/hook/... -v`
Expected: FAIL

- [ ] **Step 3: Implement Cursor hook**

Create `internal/hook/cursor.go`:

```go
package hook

import (
	"encoding/json"
	"io"
	"os"

	"github.com/longbin/agent-notify/internal/config"
	"github.com/longbin/agent-notify/internal/context"
	"github.com/longbin/agent-notify/internal/notify"
)

type cursorPayload struct {
	WorkspaceRoots []string `json:"workspace_roots"`
}

func RunCursor(r io.Reader, cfg config.Config, event string, _ io.Writer) error {
	if !cfg.EventEnabled(event) {
		return nil
	}
	var payload cursorPayload
	_ = json.NewDecoder(r).Decode(&payload)

	meta := context.MetaFromEnv("Cursor", event)
	if len(payload.WorkspaceRoots) > 0 {
		meta.CWD = payload.WorkspaceRoots[0]
	}
	title := context.Render(cfg.Notify.TitleTemplate, meta)
	body := cfg.BodyForEvent(event)
	return notify.SendAuto(cfg.Notify.Protocol, title, body)
}
```

- [ ] **Step 4: Implement Claude hook**

Create `internal/hook/claude.go`:

```go
package hook

import (
	"encoding/json"
	"io"

	"github.com/longbin/agent-notify/internal/config"
	"github.com/longbin/agent-notify/internal/context"
	"github.com/longbin/agent-notify/internal/notify"
)

type claudePayload struct {
	StopHookActive bool   `json:"stop_hook_active"`
	Message        string `json:"message"`
}

type claudeResponse struct {
	TerminalSequence string `json:"terminalSequence,omitempty"`
}

func RunClaude(r io.Reader, cfg config.Config, event string, w io.Writer) error {
	if !cfg.EventEnabled(event) {
		_, err := io.WriteString(w, "{}\n")
		return err
	}
	var payload claudePayload
	_ = json.NewDecoder(r).Decode(&payload)
	if event == "stop" && payload.StopHookActive {
		_, err := io.WriteString(w, "{}\n")
		return err
	}

	meta := context.MetaFromEnv("Claude", event)
	title := context.Render(cfg.Notify.TitleTemplate, meta)
	body := cfg.BodyForEvent(event)
	seq := notify.BuildSequence(cfg.Notify.Protocol, title, body)
	resp := claudeResponse{TerminalSequence: seq}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(resp)
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/hook/... -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/hook
git commit -m "feat: add Cursor and Claude hook handlers"
```

---

### Task 8: Install Package

**Files:**
- Create: `internal/install/install.go`
- Create: `internal/install/install_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/install/install_test.go`:

```go
package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMergeCursorHooks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hooks.json")
	existing := `{"version":1,"hooks":{"beforeShellExecution":[{"command":"other"}]}}`
	os.WriteFile(path, []byte(existing), 0644)

	if err := MergeCursorHooks(path, false); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	var doc map[string]any
	json.Unmarshal(data, &doc)
	hooks := doc["hooks"].(map[string]any)
	stop := hooks["stop"].([]any)
	if len(stop) != 1 {
		t.Fatalf("expected stop hook added")
	}
}

func TestMergeCursorHooksNoOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hooks.json")
	existing := `{"version":1,"hooks":{"stop":[{"command":"existing"}]}}`
	os.WriteFile(path, []byte(existing), 0644)
	MergeCursorHooks(path, false)
	data, _ := os.ReadFile(path)
	if string(data) != existing {
		t.Fatal("should not overwrite existing stop hook without force")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/install/... -v`
Expected: FAIL

- [ ] **Step 3: Implement install**

Create `internal/install/install.go`:

```go
package install

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/longbin/agent-notify/internal/config"
)

const cursorHookCmd = "agent-notify hook cursor stop"
const cursorToolCmd = "agent-notify hook cursor tool"
const claudeStopCmd = "agent-notify hook claude stop"
const claudeIdleCmd = "agent-notify hook claude idle"

func CursorHooksPath() string {
	return filepath.Join(os.Getenv("HOME"), ".cursor", "hooks.json")
}

func ClaudeSettingsPath() string {
	return filepath.Join(os.Getenv("HOME"), ".claude", "settings.json")
}

func MergeCursorHooks(path string, force bool) error {
	doc := map[string]any{"version": 1, "hooks": map[string]any{}}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &doc)
	}
	hooks, _ := doc["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
		doc["hooks"] = hooks
	}
	addHook(hooks, "stop", cursorHookCmd, force)
	cfg, _ := config.LoadDefault()
	if cfg.Events.Tool {
		addHook(hooks, "afterShellExecution", cursorToolCmd, force)
	}
	return writeJSON(path, doc)
}

func addHook(hooks map[string]any, name, command string, force bool) {
	entry := []any{map[string]string{"command": command}}
	if existing, ok := hooks[name]; ok && !force {
		_ = existing
		return
	}
	hooks[name] = entry
}

func MergeClaudeSettings(path string, force bool) error {
	doc := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &doc)
	}
	hooks, _ := doc["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
		doc["hooks"] = hooks
	}
	setClaudeHook(hooks, "Stop", claudeStopCmd, force)
	cfg, _ := config.LoadDefault()
	if cfg.Events.Idle {
		setClaudeHook(hooks, "Notification", claudeIdleCmd, force)
	}
	return writeJSON(path, doc)
}

func setClaudeHook(hooks map[string]any, event, command string, force bool) {
	if _, ok := hooks[event]; ok && !force {
		return
	}
	hooks[event] = []any{
		map[string]any{
			"hooks": []any{
				map[string]string{
					"type":    "command",
					"command": command,
				},
			},
		},
	}
}

func writeJSON(path string, doc any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

func InstallAll(force bool) error {
	if err := config.WriteDefault(config.DefaultPath()); err != nil {
		return err
	}
	if err := MergeCursorHooks(CursorHooksPath(), force); err != nil {
		return err
	}
	return MergeClaudeSettings(ClaudeSettingsPath(), force)
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/install/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/install
git commit -m "feat: merge Cursor and Claude hook configs on install"
```

---

### Task 9: Wire CLI Commands

**Files:**
- Modify: `cmd/agent-notify/main.go`

- [ ] **Step 1: Replace main.go with full CLI**

Replace `cmd/agent-notify/main.go` with:

```go
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/longbin/agent-notify/internal/config"
	"github.com/longbin/agent-notify/internal/context"
	"github.com/longbin/agent-notify/internal/hook"
	"github.com/longbin/agent-notify/internal/install"
	"github.com/longbin/agent-notify/internal/notify"
	"github.com/longbin/agent-notify/internal/tmux"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	if err := run(os.Args[1], os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "agent-notify:", err)
		os.Exit(1)
	}
}

func run(cmd string, args []string) error {
	switch cmd {
	case "send":
		return cmdSend(args)
	case "hook":
		return cmdHook(args)
	case "install":
		return cmdInstall(args)
	case "test":
		return cmdTest()
	case "doctor":
		return cmdDoctor()
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		printUsage()
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func cmdSend(args []string) error {
	fs := flag.NewFlagSet("send", flag.ExitOnError)
	title := fs.String("title", "", "notification title")
	body := fs.String("body", "", "notification body")
	event := fs.String("event", "stop", "event type")
	_ = fs.Parse(args)
	cfg, err := config.LoadDefault()
	if err != nil {
		return err
	}
	if *title == "" {
		meta := context.MetaFromEnv("Agent", *event)
		*title = context.Render(cfg.Notify.TitleTemplate, meta)
	}
	if *body == "" {
		*body = cfg.BodyForEvent(*event)
	}
	return notify.SendAuto(cfg.Notify.Protocol, *title, *body)
}

func cmdHook(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: agent-notify hook <cursor|claude> <event>")
	}
	agent, event := args[0], args[1]
	cfg, err := config.LoadDefault()
	if err != nil {
		return err
	}
	switch agent {
	case "cursor":
		return hook.RunCursor(os.Stdin, cfg, event, os.Stdout)
	case "claude":
		return hook.RunClaude(os.Stdin, cfg, event, os.Stdout)
	default:
		return fmt.Errorf("unknown agent %q", agent)
	}
}

func cmdInstall(args []string) error {
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	all := fs.Bool("all", false, "install all")
	force := fs.Bool("force", false, "overwrite existing hooks")
	_ = fs.Parse(args)
	if *all || len(args) == 0 {
		return install.InstallAll(*force)
	}
	return fmt.Errorf("use --all")
}

func cmdTest() error {
	return notify.TestNotification("agent-notify", "测试通知 — 如果你看到这条，说明配置正确")
}

func cmdDoctor() error {
	fmt.Println("agent-notify doctor")
	if tmux.InTmux() {
		fmt.Println("✓ running inside tmux")
		ok, val, err := tmux.AllowPassthroughEnabled()
		if err != nil {
			fmt.Printf("✗ allow-passthrough check failed: %v\n", err)
		} else if ok {
			fmt.Printf("✓ allow-passthrough=%s\n", val)
		} else {
			fmt.Printf("✗ allow-passthrough=%q — add to ~/.tmux.conf: set -g allow-passthrough on\n", val)
		}
		tty, _ := tmux.ClientTTY()
		fmt.Printf("  client_tty=%s\n", tty)
	} else {
		fmt.Println("  not in tmux (direct Ghostty mode)")
	}
	cfgPath := config.DefaultPath()
	fmt.Printf("  config=%s\n", cfgPath)
	return nil
}

func printUsage() {
	fmt.Fprint(os.Stderr, `Usage: agent-notify <command>
Commands:
  send [--title T] [--body B] [--event stop|idle|tool]
  hook cursor stop|tool
  hook claude stop|idle
  install [--all] [--force]
  test
  doctor
`)
}
```

- [ ] **Step 2: Build and smoke test**

Run:
```bash
go build -o bin/agent-notify ./cmd/agent-notify
./bin/agent-notify doctor
./bin/agent-notify test
```
Expected: doctor prints status; test sends OSC (visible in Ghostty)

- [ ] **Step 3: Run all tests**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/agent-notify/main.go
git commit -m "feat: wire send, hook, install, test, and doctor commands"
```

---

### Task 10: README and Manual Integration Tests

**Files:**
- Create: `README.md`

- [ ] **Step 1: Write README**

Create `README.md` covering:
- 安装：`make install`
- tmux 配置：`set -g allow-passthrough on`
- Hook 安装：`agent-notify install --all`
- 手动测试矩阵（无 tmux / 本地 tmux / SSH 远程 tmux）
- Cursor CLI 仅支持 `cursor-agent`，不支持 Cursor IDE
- Claude `terminalSequence` 机制说明

- [ ] **Step 2: Manual verification checklist**

Run each and confirm Ghostty notification:

```bash
# 1. 无 tmux
agent-notify test

# 2. 本地 tmux
tmux new-session -d 'agent-notify test'

# 3. install + doctor
agent-notify install --all
agent-notify doctor
```

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: add README with setup and manual test matrix"
```

---

## Spec Coverage Check

| Spec requirement | Task |
|------------------|------|
| OSC 777/9 | Task 3 |
| tmux client_tty + DCS passthrough | Task 4, 5 |
| Config TOML + event toggles | Task 2 |
| Title `{agent} — {context}` | Task 6 |
| Cursor CLI stop/tool hooks | Task 7, 8, 9 |
| Claude Stop/Notification + terminalSequence | Task 7, 8, 9 |
| Claude stop_hook_active guard | Task 7 |
| install --all merge hooks | Task 8, 9 |
| doctor + test commands | Task 9 |
| No Cursor IDE support | Task 10 README |
| No notify-send fallback | Out of scope (documented in spec) |

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-26-agent-notify.md`.

**Two execution options:**

1. **Subagent-Driven (recommended)** — 每个 Task 派发独立 subagent，任务间做 review，迭代快
2. **Inline Execution** — 在本会话用 executing-plans 批量执行，checkpoint 处暂停 review

**Which approach?**
