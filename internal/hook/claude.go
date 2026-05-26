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
