package hook

import (
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func debouncePath() string {
	return filepath.Join(os.Getenv("HOME"), ".local", "state", "agent-notify", "debounce")
}

func shouldDebounce(title string) bool {
	path := debouncePath()
	data, err := os.ReadFile(path)
	now := time.Now().UnixNano()
	window := int64(2 * time.Second)

	var lastTitle string
	var lastAt int64
	if err == nil {
		// format: timestamp\ttitle
		for i := 0; i < len(data); i++ {
			if data[i] == '\t' {
				lastAt, _ = strconv.ParseInt(string(data[:i]), 10, 64)
				lastTitle = string(data[i+1:])
				break
			}
		}
	}

	if lastTitle == title && now-lastAt < window {
		return true
	}

	_ = os.MkdirAll(filepath.Dir(path), 0755)
	_ = os.WriteFile(path, []byte(strconv.FormatInt(now, 10)+"\t"+title), 0644)
	return false
}
