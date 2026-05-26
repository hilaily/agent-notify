package hook

import (
	"context"
	"time"

	"github.com/longbin/agent-notify/internal/config"
	"github.com/longbin/agent-notify/internal/inbox"
	"github.com/longbin/agent-notify/internal/logx"
)

var (
	uploadInboxRecord = defaultUploadInboxRecord
	appendInboxRecord = func(rec inbox.Record) error {
		return inbox.NewStore("").Append(rec)
	}
)

func recordInbox(cfg config.Config, agent, event, cwd, title, body string) {
	if !cfg.Inbox.Enabled {
		return
	}
	rec := inbox.BuildRecord(inbox.BuildInput{
		Agent:  agent,
		Event:  event,
		CWD:    cwd,
		Title:  title,
		Body:   body,
		Source: inbox.SourceRemote,
	})
	if err := uploadInboxRecord(rec, cfg); err != nil {
		logx.Append("inbox upload failed: %v", err)
		if !cfg.Inbox.FallbackLocal {
			return
		}
		rec.Source = inbox.SourceFallback
		if err := appendInboxRecord(rec); err != nil {
			logx.Append("inbox fallback append failed: %v", err)
		}
	}
}

func defaultUploadInboxRecord(rec inbox.Record, cfg config.Config) error {
	timeout := time.Duration(cfg.Inbox.TimeoutMS) * time.Millisecond
	client := inbox.NewClient(inbox.ClientConfig{
		Socket:  cfg.Inbox.RemoteSocket,
		Addr:    cfg.Inbox.Addr,
		Timeout: timeout,
	})
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return client.Upload(ctx, rec)
}
