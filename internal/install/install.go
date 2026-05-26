package install

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/longbin/agent-notify/internal/config"
)

const cursorStopCmd = "agent-notify hook cursor stop"
const cursorResponseCmd = "agent-notify hook cursor response"
const cursorToolCmd = "agent-notify hook cursor tool"
const claudeStopCmd = "agent-notify hook claude stop"
const claudeIdleCmd = "agent-notify hook claude idle"

func CursorHooksPath() string {
	return filepath.Join(os.Getenv("HOME"), ".cursor", "hooks.json")
}

func ClaudeSettingsPath() string {
	return filepath.Join(os.Getenv("HOME"), ".claude", "settings.json")
}

func MergeCursorHooks(path string, force bool) error {
	doc := map[string]any{"version": 1, "hooks": map[string]any{}}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &doc)
	}
	hooks, _ := doc["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
		doc["hooks"] = hooks
	}
	addHook(hooks, "stop", cursorStopCmd, force)
	addHook(hooks, "afterAgentResponse", cursorResponseCmd, force)
	cfg, _ := config.LoadDefault()
	if cfg.Events.Tool {
		addHook(hooks, "afterShellExecution", cursorToolCmd, force)
	}
	return writeJSON(path, doc)
}

func addHook(hooks map[string]any, name, command string, force bool) {
	if existing, ok := hooks[name]; ok && !force {
		_ = existing
		return
	}
	hooks[name] = []any{map[string]string{"command": command}}
}

func MergeClaudeSettings(path string, force bool) error {
	doc := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &doc)
	}
	hooks, _ := doc["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
		doc["hooks"] = hooks
	}
	setClaudeHook(hooks, "Stop", claudeStopCmd, force)
	cfg, _ := config.LoadDefault()
	if cfg.Events.Idle {
		setClaudeHook(hooks, "Notification", claudeIdleCmd, force)
	}
	return writeJSON(path, doc)
}

func setClaudeHook(hooks map[string]any, event, command string, force bool) {
	if _, ok := hooks[event]; ok && !force {
		return
	}
	hooks[event] = []any{
		map[string]any{
			"hooks": []any{
				map[string]string{
					"type":    "command",
					"command": command,
				},
			},
		},
	}
}

func writeJSON(path string, doc any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

func InstallAll(force bool) error {
	if err := config.Default().WriteDefault(config.DefaultPath()); err != nil {
		return err
	}
	if err := MergeCursorHooks(CursorHooksPath(), force); err != nil {
		return err
	}
	return MergeClaudeSettings(ClaudeSettingsPath(), force)
}
