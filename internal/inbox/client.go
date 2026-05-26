package inbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

type ClientConfig struct {
	URL     string
	Socket  string
	Addr    string
	Timeout time.Duration
}

type Client struct {
	cfg ClientConfig
}

func NewClient(cfg ClientConfig) Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 500 * time.Millisecond
	}
	return Client{cfg: cfg}
}

func (c Client) Upload(ctx context.Context, rec Record) error {
	body, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	url := c.url()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url+"/inbox", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: c.cfg.Timeout}
	if c.cfg.Socket != "" {
		socket := c.cfg.Socket
		client.Transport = &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", socket)
			},
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("inbox upload failed: %s", resp.Status)
	}
	return nil
}

func (c Client) url() string {
	if c.cfg.URL != "" {
		return strings.TrimRight(c.cfg.URL, "/")
	}
	if c.cfg.Socket != "" {
		return "http://unix"
	}
	if c.cfg.Addr != "" {
		return "http://" + c.cfg.Addr
	}
	return "http://127.0.0.1:17777"
}
