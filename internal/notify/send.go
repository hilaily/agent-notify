package notify

import (
	"fmt"
	"io"
	"os"

	"github.com/longbin/agent-notify/internal/tmux"
)

type DeliveryMethod string

const (
	MethodDirectStdout             DeliveryMethod = "direct-stdout"
	MethodPassthroughStdout        DeliveryMethod = "passthrough-stdout"
	MethodClientTTYRaw             DeliveryMethod = "client-tty-raw"
	MethodClientTTYPassthrough     DeliveryMethod = "client-tty-passthrough"
	MethodControllingTTYRaw        DeliveryMethod = "controlling-tty-raw"
	MethodControllingTTYPassthrough DeliveryMethod = "controlling-tty-passthrough"
)

type SendOptions struct {
	Protocol  string
	Title     string
	Body      string
	Writer    io.Writer
	InTmux    bool
	Layers    int
	ClientTTY string
	Method    DeliveryMethod
	ForHook   bool
}

type SendResult struct {
	Method DeliveryMethod
}

func autoMethods(inTmux, ssh bool) []DeliveryMethod {
	if !inTmux {
		return []DeliveryMethod{MethodDirectStdout}
	}
	if ssh {
		return []DeliveryMethod{
			MethodPassthroughStdout,
			MethodClientTTYPassthrough,
			MethodClientTTYRaw,
		}
	}
	return []DeliveryMethod{
		MethodClientTTYRaw,
		MethodPassthroughStdout,
		MethodClientTTYPassthrough,
	}
}

// hookMethods avoids stdout when Cursor captures hook output (pipe, not a TTY).
func hookMethods(inTmux, ssh bool, clientTTY string) []DeliveryMethod {
	var methods []DeliveryMethod
	methods = append(methods, MethodControllingTTYRaw, MethodControllingTTYPassthrough)
	if inTmux && clientTTY != "" {
		if ssh {
			methods = append(methods, MethodClientTTYRaw, MethodClientTTYPassthrough)
		} else {
			methods = append(methods, MethodClientTTYRaw, MethodClientTTYPassthrough)
		}
	}
	if stdoutIsTerminal() {
		if !inTmux {
			methods = append(methods, MethodDirectStdout)
		} else {
			methods = append(methods, MethodPassthroughStdout)
		}
	}
	return methods
}

func stdoutIsTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func Send(opts SendOptions) error {
	_, err := SendWithResult(opts)
	return err
}

func SendWithResult(opts SendOptions) (SendResult, error) {
	seq := BuildSequence(opts.Protocol, opts.Title, opts.Body)
	methods := []DeliveryMethod{opts.Method}
	if opts.Method == "" {
		if opts.ForHook {
			methods = hookMethods(opts.InTmux, tmux.IsSSHSession(), opts.ClientTTY)
		} else {
			methods = autoMethods(opts.InTmux, tmux.IsSSHSession())
		}
	}

	var lastErr error
	for _, method := range methods {
		if err := deliver(seq, opts, method); err != nil {
			lastErr = err
			continue
		}
		return SendResult{Method: method}, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no delivery method available")
	}
	return SendResult{}, lastErr
}

func deliver(seq string, opts SendOptions, method DeliveryMethod) error {
	switch method {
	case MethodDirectStdout:
		return writeOut(seq, opts, seq)
	case MethodPassthroughStdout:
		layers := opts.Layers
		if layers <= 0 {
			layers = 1
		}
		return writeOut(seq, opts, tmux.WrapPassthroughLayers(seq, layers))
	case MethodClientTTYRaw:
		return writeClientTTY(opts.ClientTTY, seq)
	case MethodClientTTYPassthrough:
		layers := opts.Layers
		if layers <= 0 {
			layers = 1
		}
		return writeClientTTY(opts.ClientTTY, tmux.WrapPassthroughLayers(seq, layers))
	case MethodControllingTTYRaw:
		return writeControllingTTY(seq)
	case MethodControllingTTYPassthrough:
		layers := opts.Layers
		if layers <= 0 {
			layers = 1
		}
		return writeControllingTTY(tmux.WrapPassthroughLayers(seq, layers))
	default:
		return fmt.Errorf("unknown delivery method %q", method)
	}
}

func writeOut(_ string, opts SendOptions, out string) error {
	w := opts.Writer
	if w == nil {
		w = os.Stdout
	}
	_, err := io.WriteString(w, out)
	return err
}

func writeClientTTY(clientTTY, out string) error {
	if clientTTY == "" {
		return fmt.Errorf("client tty unavailable")
	}
	f, err := os.OpenFile(clientTTY, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.WriteString(f, out)
	return err
}

func writeControllingTTY(out string) error {
	f, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.WriteString(f, out)
	return err
}

func SendForHookWithResult(protocol, title, body string) (SendResult, error) {
	inTmux := tmux.InTmux()
	clientTTY, _ := tmux.ClientTTY()
	layers := 0
	if inTmux {
		layers = 1
	}
	return SendWithResult(SendOptions{
		Protocol:  protocol,
		Title:     title,
		Body:      body,
		InTmux:    inTmux,
		Layers:    layers,
		ClientTTY: clientTTY,
		ForHook:   true,
	})
}

func SendAuto(protocol, title, body string) error {
	_, err := SendAutoWithResult(protocol, title, body)
	return err
}

func SendAutoWithResult(protocol, title, body string) (SendResult, error) {
	inTmux := tmux.InTmux()
	clientTTY, _ := tmux.ClientTTY()
	layers := 0
	if inTmux {
		layers = 1
	}
	return SendWithResult(SendOptions{
		Protocol:  protocol,
		Title:     title,
		Body:      body,
		InTmux:    inTmux,
		Layers:    layers,
		ClientTTY: clientTTY,
	})
}

func EmitSequence(seq string) (SendResult, error) {
	inTmux := tmux.InTmux()
	clientTTY, _ := tmux.ClientTTY()
	layers := 0
	if inTmux {
		layers = 1
	}
	opts := SendOptions{
		InTmux:    inTmux,
		Layers:    layers,
		ClientTTY: clientTTY,
	}
	methods := autoMethods(inTmux, tmux.IsSSHSession())
	var lastErr error
	for _, method := range methods {
		if err := deliver(seq, opts, method); err != nil {
			lastErr = err
			continue
		}
		return SendResult{Method: method}, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no delivery method available")
	}
	return SendResult{}, lastErr
}
