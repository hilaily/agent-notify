package tmux

import (
	"os"
	"os/exec"
	"strings"
)

func InTmux() bool {
	return os.Getenv("TMUX") != ""
}

func ClientTTY() (string, error) {
	if !InTmux() {
		return "", nil
	}
	out, err := exec.Command("tmux", "display-message", "-p", "#{client_tty}").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func WrapPassthrough(seq string) string {
	return "\033Ptmux;\033" + seq + "\033\\"
}

func WrapPassthroughLayers(seq string, layers int) string {
	out := seq
	for i := 0; i < layers; i++ {
		out = WrapPassthrough(out)
	}
	return out
}

func AllowPassthroughEnabled() (bool, string, error) {
	if !InTmux() {
		return true, "", nil
	}
	out, err := exec.Command("tmux", "show-option", "-gv", "allow-passthrough").Output()
	if err != nil {
		return false, "", err
	}
	val := strings.TrimSpace(string(out))
	return val == "on" || val == "all", val, nil
}
