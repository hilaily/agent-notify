package hook

import (
	"encoding/json"
	"io"

	"github.com/longbin/agent-notify/internal/config"
	"github.com/longbin/agent-notify/internal/context"
	"github.com/longbin/agent-notify/internal/logx"
	"github.com/longbin/agent-notify/internal/notify"
)

type claudePayload struct {
	StopHookActive bool   `json:"stop_hook_active"`
	Message        string `json:"message"`
	HookEventName  string `json:"hook_event_name"`
}

type claudeResponse struct {
	TerminalSequence string `json:"terminalSequence,omitempty"`
}

func RunClaude(r io.Reader, cfg config.Config, event string, w io.Writer) error {
	var payload claudePayload
	decodeJSON(r, &payload)
	hookName := payload.HookEventName
	if hookName == "" {
		hookName = event
	}
	logx.Append("hook claude event=%s hook_event_name=%s enabled=%v stop_hook_active=%v",
		event, hookName, cfg.EventEnabled(event), payload.StopHookActive)

	if !cfg.EventEnabled(event) {
		logx.Append("hook claude event=%s skipped (disabled in config)", event)
		_, err := io.WriteString(w, "{}\n")
		return err
	}
	if event == "stop" && payload.StopHookActive {
		logx.Append("hook claude event=stop skipped (stop_hook_active)")
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
	if err := enc.Encode(resp); err != nil {
		logx.Append("hook claude event=%s encode FAILED: %v", event, err)
		return err
	}
	logx.Append("hook claude event=%s terminalSequence OK title=%q", event, title)
	recordInbox(cfg, "Claude", event, meta.CWD, title, body)
	return nil
}
