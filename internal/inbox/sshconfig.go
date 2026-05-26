package inbox

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	sshConfigBegin = "# BEGIN agent-notify inbox"
	sshConfigEnd   = "# END agent-notify inbox"
)

func DefaultSSHConfigPath() string {
	return filepath.Join(os.Getenv("HOME"), ".ssh", "config")
}

func SSHConfigBlock(remoteSocket, localSocket string) string {
	return strings.Join([]string{
		sshConfigBegin,
		"Host *",
		"    RemoteForward " + remoteSocket + " " + localSocket,
		"    ExitOnForwardFailure no",
		"    ServerAliveInterval 30",
		sshConfigEnd,
		"",
	}, "\n")
}

func InstallSSHConfig(path, remoteSocket, localSocket string) (string, error) {
	if path == "" {
		path = DefaultSSHConfigPath()
	}
	block := SSHConfigBlock(remoteSocket, localSocket)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return "", err
	}

	var current string
	if data, err := os.ReadFile(path); err == nil {
		current = string(data)
	} else if !os.IsNotExist(err) {
		return "", err
	}

	next := replaceManagedBlock(current, block)
	if err := os.WriteFile(path, []byte(next), 0600); err != nil {
		return "", err
	}
	return block, nil
}

func replaceManagedBlock(current, block string) string {
	start := strings.Index(current, sshConfigBegin)
	end := strings.Index(current, sshConfigEnd)
	if start >= 0 && end >= start {
		end += len(sshConfigEnd)
		for end < len(current) && (current[end] == '\n' || current[end] == '\r') {
			end++
		}
		prefix := strings.TrimRight(current[:start], "\n")
		suffix := strings.TrimLeft(current[end:], "\n")
		var parts []string
		if prefix != "" {
			parts = append(parts, prefix)
		}
		parts = append(parts, strings.TrimRight(block, "\n"))
		if suffix != "" {
			parts = append(parts, suffix)
		}
		return strings.Join(parts, "\n\n") + "\n"
	}
	if strings.TrimSpace(current) == "" {
		return block
	}
	return strings.TrimRight(current, "\n") + "\n\n" + block
}
