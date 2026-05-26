package inbox

import (
	"os"
	"time"
)

type BuildInput struct {
	Agent  string
	Event  string
	CWD    string
	Title  string
	Body   string
	Source string
}

func BuildRecord(input BuildInput) Record {
	host, _ := os.Hostname()
	source := input.Source
	if source == "" {
		source = SourceLocal
	}
	return Record{
		ID:     NewID(time.Now()),
		Time:   time.Now(),
		Host:   host,
		Agent:  input.Agent,
		Event:  input.Event,
		CWD:    input.CWD,
		Title:  input.Title,
		Body:   input.Body,
		Status: StatusPending,
		Source: source,
		Tmux: TmuxContext{
			Pane: os.Getenv("TMUX_PANE"),
		},
	}
}
