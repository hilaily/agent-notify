package logx

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func Path() string {
	return filepath.Join(os.Getenv("HOME"), ".local", "state", "agent-notify", "hook.log")
}

func Append(format string, args ...any) {
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(f, "[%s] %s\n", time.Now().Format(time.RFC3339), msg)
}

func Tail(n int) ([]string, error) {
	data, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	lines := splitLines(string(data))
	if n <= 0 || n >= len(lines) {
		return lines, nil
	}
	return lines[len(lines)-n:], nil
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if line != "" {
				lines = append(lines, line)
			}
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
