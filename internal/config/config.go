package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Events Events `toml:"events"`
	Notify Notify `toml:"notify"`
	Inbox  Inbox  `toml:"inbox"`
}

type Events struct {
	Stop     bool `toml:"stop"`
	Response bool `toml:"response"`
	Idle     bool `toml:"idle"`
	Tool     bool `toml:"tool"`
}

type Notify struct {
	Protocol      string `toml:"protocol"`
	TitleTemplate string `toml:"title_template"`
	BodyStop      string `toml:"body_stop"`
	BodyIdle      string `toml:"body_idle"`
	BodyTool      string `toml:"body_tool"`
}

type Inbox struct {
	Enabled       bool   `toml:"enabled"`
	Socket        string `toml:"socket"`
	RemoteSocket  string `toml:"remote_socket"`
	Addr          string `toml:"addr"`
	FallbackLocal bool   `toml:"fallback_local"`
	TimeoutMS     int    `toml:"timeout_ms"`
}

func Default() Config {
	return Config{
		Events: Events{Stop: true, Response: true, Idle: false, Tool: false},
		Notify: Notify{
			Protocol:      "osc777",
			TitleTemplate: "{agent} — {context}",
			BodyStop:      "等待输入",
			BodyIdle:      "空闲 60s+，等待输入",
			BodyTool:      "工具执行完成",
		},
		Inbox: Inbox{
			Enabled:       true,
			Socket:        DefaultInboxSocket(),
			RemoteSocket:  DefaultInboxRemoteSocket(),
			Addr:          "127.0.0.1:17777",
			FallbackLocal: true,
			TimeoutMS:     500,
		},
	}
}

func DefaultInboxRemoteSocket() string {
	user := os.Getenv("USER")
	if user == "" {
		user = "user"
	}
	return filepath.Join("/tmp", "agent-notify-"+user+".sock")
}

func DefaultInboxSocket() string {
	if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
		return filepath.Join(runtimeDir, "agent-notify.sock")
	}
	return filepath.Join(os.Getenv("HOME"), ".local", "state", "agent-notify", "agent-notify.sock")
}

func DefaultPath() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "agent-notify", "config.toml")
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func LoadDefault() (Config, error) {
	return Load(DefaultPath())
}

func (c Config) EventEnabled(event string) bool {
	switch strings.ToLower(event) {
	case "stop":
		return c.Events.Stop
	case "response":
		return c.Events.Response
	case "idle":
		return c.Events.Idle
	case "tool":
		return c.Events.Tool
	default:
		return false
	}
}

func (c Config) BodyForEvent(event string) string {
	switch strings.ToLower(event) {
	case "idle":
		return c.Notify.BodyIdle
	case "tool":
		return c.Notify.BodyTool
	default:
		return c.Notify.BodyStop
	}
}

func (c Config) WriteDefault(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(Default())
}
