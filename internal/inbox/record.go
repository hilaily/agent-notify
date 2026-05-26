package inbox

import "time"

const (
	StatusPending = "pending"
	StatusDone    = "done"
)

const (
	SourceLocal    = "local"
	SourceRemote   = "remote"
	SourceFallback = "fallback"
)

type TmuxContext struct {
	Session string `json:"session,omitempty"`
	Window  string `json:"window,omitempty"`
	Pane    string `json:"pane,omitempty"`
}

type Record struct {
	ID     string      `json:"id"`
	Time   time.Time   `json:"time"`
	Host   string      `json:"host,omitempty"`
	Agent  string      `json:"agent,omitempty"`
	Event  string      `json:"event,omitempty"`
	CWD    string      `json:"cwd,omitempty"`
	Title  string      `json:"title,omitempty"`
	Body   string      `json:"body,omitempty"`
	Status string      `json:"status"`
	Source string      `json:"source,omitempty"`
	Tmux   TmuxContext `json:"tmux,omitempty"`
}
