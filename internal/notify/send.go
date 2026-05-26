package notify

import (
	"fmt"
	"io"
	"os"

	"github.com/longbin/agent-notify/internal/tmux"
)

type SendOptions struct {
	Protocol  string
	Title     string
	Body      string
	Writer    io.Writer
	InTmux    bool
	Layers    int
	ClientTTY string
}

func Send(opts SendOptions) error {
	seq := BuildSequence(opts.Protocol, opts.Title, opts.Body)
	w := opts.Writer
	if w == nil {
		w = os.Stdout
	}

	if opts.InTmux && opts.ClientTTY != "" {
		f, err := os.OpenFile(opts.ClientTTY, os.O_WRONLY, 0)
		if err == nil {
			defer f.Close()
			_, err = io.WriteString(f, seq)
			return err
		}
	}

	out := seq
	if opts.InTmux {
		layers := opts.Layers
		if layers <= 0 {
			layers = 1
		}
		out = tmux.WrapPassthroughLayers(seq, layers)
	}
	_, err := io.WriteString(w, out)
	return err
}

func SendAuto(protocol, title, body string) error {
	inTmux := tmux.InTmux()
	clientTTY, _ := tmux.ClientTTY()
	layers := 0
	if inTmux {
		layers = 1
	}
	return Send(SendOptions{
		Protocol:  protocol,
		Title:     title,
		Body:      body,
		InTmux:    inTmux,
		Layers:    layers,
		ClientTTY: clientTTY,
	})
}

func TestNotification(title, body string) error {
	if err := SendAuto("osc777", title, body); err != nil {
		return fmt.Errorf("send test notification: %w", err)
	}
	return nil
}
