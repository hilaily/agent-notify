package notify

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/longbin/agent-notify/internal/tmux"
)

const defaultTestTitle = "agent-notify"
const defaultTestBody = "测试通知 — 如果你看到这条，说明配置正确"

type claudeTestResponse struct {
	TerminalSequence string `json:"terminalSequence"`
}

func TestCursor(title, body string, tryAll bool) (SendResult, error) {
	if title == "" {
		title = defaultTestTitle
	}
	if body == "" {
		body = defaultTestBody + " [cursor]"
	}

	if tryAll {
		return testCursorAll(title, body)
	}

	result, err := SendAutoWithResult("osc777", title, body)
	printTestStatus("cursor", result, err)
	return result, err
}

func testCursorAll(title, body string) (SendResult, error) {
	inTmux := tmux.InTmux()
	methods := []DeliveryMethod{MethodDirectStdout, MethodPassthroughStdout, MethodClientTTYRaw, MethodClientTTYPassthrough}
	if !inTmux {
		methods = []DeliveryMethod{MethodDirectStdout}
	}
	clientTTY, _ := tmux.ClientTTY()

	var lastResult SendResult
	var lastErr error
	for i, method := range methods {
		result, err := SendWithResult(SendOptions{
			Protocol:  "osc777",
			Title:     title,
			Body:      fmt.Sprintf("%s [%s]", body, method),
			InTmux:    inTmux,
			Layers:    1,
			ClientTTY: clientTTY,
			Method:    method,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "try %d/%d %s: FAIL %v\n", i+1, len(methods), method, err)
			lastErr = err
			continue
		}
		fmt.Fprintf(os.Stderr, "try %d/%d %s: OK (check Ghostty notification)\n", i+1, len(methods), method)
		lastResult = result
	}
	if lastResult.Method == "" && lastErr != nil {
		return lastResult, lastErr
	}
	fmt.Fprintf(os.Stderr, "mode=cursor try-all done ssh=%v tmux=%v\n", tmux.IsSSHSession(), inTmux)
	return lastResult, nil
}

func printTestStatus(mode string, result SendResult, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "agent-notify: %s test FAILED: %v\n", mode, err)
		return
	}
	fmt.Fprintf(os.Stderr,
		"agent-notify: %s test sent via %s (ssh=%v tmux=%v) — check Ghostty desktop notification\n",
		mode, result.Method, tmux.IsSSHSession(), tmux.InTmux(),
	)
}

func TestClaude(title, body string, apply bool, out io.Writer) (SendResult, error) {
	if title == "" {
		title = defaultTestTitle
	}
	if body == "" {
		body = defaultTestBody + " [claude]"
	}
	seq := BuildSequence("osc777", title, body)
	if out == nil {
		out = os.Stdout
	}
	if !apply {
		resp := claudeTestResponse{TerminalSequence: seq}
		enc := json.NewEncoder(out)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(resp); err != nil {
			return SendResult{}, err
		}
		fmt.Fprintln(os.Stderr, "agent-notify: claude test JSON printed (use --apply to emit)")
		return SendResult{Method: "terminal-sequence-json"}, nil
	}
	result, err := EmitSequence(seq)
	printTestStatus("claude", result, err)
	return result, err
}

func TestNotification(title, body string) error {
	_, err := TestCursor(title, body, false)
	return err
}
