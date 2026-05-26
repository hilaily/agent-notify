package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMergeCursorHooks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hooks.json")
	existing := `{"version":1,"hooks":{"beforeShellExecution":[{"command":"other"}]}}`
	os.WriteFile(path, []byte(existing), 0644)

	if err := MergeCursorHooks(path, false); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	var doc map[string]any
	json.Unmarshal(data, &doc)
	hooks := doc["hooks"].(map[string]any)
	stop := hooks["stop"].([]any)
	if len(stop) != 1 {
		t.Fatalf("expected stop hook added")
	}
	resp := hooks["afterAgentResponse"].([]any)
	if len(resp) != 1 {
		t.Fatalf("expected afterAgentResponse hook added")
	}
}

func TestMergeCursorHooksNoOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hooks.json")
	existing := `{"version":1,"hooks":{"stop":[{"command":"existing"}]}}`
	os.WriteFile(path, []byte(existing), 0644)
	MergeCursorHooks(path, false)
	data, _ := os.ReadFile(path)
	var doc map[string]any
	json.Unmarshal(data, &doc)
	hooks := doc["hooks"].(map[string]any)
	stop := hooks["stop"].([]any)
	entry := stop[0].(map[string]any)
	if entry["command"] != "existing" {
		t.Fatal("should not overwrite existing stop hook without force")
	}
}
