package context

import (
	"os"
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
	ctx := m.ResolveContext()
	out := strings.ReplaceAll(tmpl, "{agent}", m.Agent)
	out = strings.ReplaceAll(out, "{context}", ctx)
	return out
}

func (m Meta) ResolveContext() string {
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
		Agent: agent,
		CWD:   cwd,
		Event: event,
	}
}
