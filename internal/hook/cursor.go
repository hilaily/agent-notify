package hook

import (
	"io"
	"os"

	"github.com/longbin/agent-notify/internal/config"
	"github.com/longbin/agent-notify/internal/context"
	"github.com/longbin/agent-notify/internal/logx"
	"github.com/longbin/agent-notify/internal/notify"
	"github.com/longbin/agent-notify/internal/tmux"
)

type cursorPayload struct {
	WorkspaceRoots []string `json:"workspace_roots"`
	HookEventName  string   `json:"hook_event_name"`
	Status         string   `json:"status"`
}

var (
	sendForHook = notify.SendForHookWithResult
	sendAuto    = notify.SendAutoWithResult
)

func RunCursor(r io.Reader, cfg config.Config, event string, _ io.Writer) error {
	var payload cursorPayload
	fromHook := !isInteractiveStdin(r)
	decodeJSON(r, &payload)

	hookName := payload.HookEventName
	if hookName == "" {
		hookName = event
	}
	clientTTY, _ := tmux.ClientTTY()
	logx.Append("hook cursor event=%s hook_event_name=%s status=%s enabled=%v from_hook=%v stdout_tty=%v client_tty=%q",
		event, hookName, payload.Status, cfg.EventEnabled(event), fromHook, stdoutIsTerminal(), clientTTY)

	if !cfg.EventEnabled(event) {
		logx.Append("hook cursor event=%s skipped (disabled in config)", event)
		return nil
	}

	meta := context.MetaFromEnv("Cursor", event)
	if len(payload.WorkspaceRoots) > 0 {
		meta.CWD = payload.WorkspaceRoots[0]
	}
	title := context.Render(cfg.Notify.TitleTemplate, meta)
	body := cfg.BodyForEvent(event)

	if shouldDebounce(title) {
		logx.Append("hook cursor event=%s skipped (debounced duplicate)", event)
		return nil
	}

	var result notify.SendResult
	var err error
	if fromHook {
		result, err = sendForHook(cfg.Notify.Protocol, title, body)
	} else {
		result, err = sendAuto(cfg.Notify.Protocol, title, body)
	}
	if err != nil {
		logx.Append("hook cursor event=%s send FAILED: %v", event, err)
		return err
	}
	logx.Append("hook cursor event=%s send OK via %s title=%q", event, result.Method, title)
	recordInbox(cfg, "Cursor", event, meta.CWD, title, body)
	return nil
}

func stdoutIsTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
