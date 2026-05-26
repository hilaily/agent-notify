package hook

import (
	"encoding/json"
	"io"

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
