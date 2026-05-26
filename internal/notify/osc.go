package notify

import "strings"

func BuildSequence(protocol, title, body string) string {
	switch protocol {
	case "osc9":
		return "\033]9;" + escapeOSCField(body) + "\007"
	default:
		return "\033]777;notify;" + escapeOSCField(title) + ";" + escapeOSCField(body) + "\007"
	}
}

func escapeOSCField(s string) string {
	return strings.ReplaceAll(s, ";", "\\;")
}
