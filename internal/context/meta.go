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
