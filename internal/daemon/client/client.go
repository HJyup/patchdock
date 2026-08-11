package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/HJyup/patchdock/internal/daemon/api"
	"github.com/HJyup/patchdock/internal/transport"
)

const (
	baseURL        = "http://patchdock"
	defaultTimeout = 5 * time.Second
)

type Client struct {
	socket  string
	timeout time.Duration
	http    *http.Client
}

func New(socket string) *Client {
	c := &Client{
		socket:  socket,
		timeout: defaultTimeout,
	}

	c.http = &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return transport.Dial(ctx, c.socket)
			},
		},
	}

	return c
}

func (c *Client) Health(ctx context.Context) (api.HealthResponse, error) {
	path := "/health"

	var out api.HealthResponse
	err := c.do(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

func (c *Client) Run(ctx context.Context, paylod api.RunRequest) (api.RunResponse, error) {
	path := "/run"

	var out api.RunResponse
	err := c.do(ctx, http.MethodPost, path, paylod, &out)
	return out, err
}

func (c *Client) Cancel(ctx context.Context, runID string) error {
	path := "/run/" + url.PathEscape(runID)

	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) StreamRuns(ctx context.Context, handler func(snapshot api.Snapshot) error) error {
	path := "/run"

	return c.Stream(ctx, path, handler)
}

func (c *Client) Stream(ctx context.Context, path string, handler func(snapshot api.Snapshot) error) error {
	req, err := c.newStreamRequest(ctx, path)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return DeamonError(err)
	}
	defer resp.Body.Close()

	if err := checkStatus(resp); err != nil {
		return err
	}

	scanner := bufio.NewScanner(resp.Body)
	var eventType string

	for scanner.Scan() {
		line := scanner.Bytes()

		if value, ok := bytes.CutPrefix(line, []byte("event: ")); ok {
			eventType = string(bytes.TrimSpace(value))
			continue
		}

		if data, ok := bytes.CutPrefix(line, []byte("data: ")); ok {
			switch eventType {
			case api.EventError:
				var e api.ErrorEvent

				if err := json.Unmarshal(data, &e); err != nil {
					return fmt.Errorf("unmarshal error event data: %w", err)
				}

				return fmt.Errorf("daemon stream: %s", e.Message)

			case api.EventSnapshot:
				var snapshot api.Snapshot

				if err := json.Unmarshal(data, &snapshot); err != nil {
					return fmt.Errorf("unmarshal snapshot event data: %w", err)
				}

				if err := handler(snapshot); err != nil {
					return err
				}
			}

			eventType = ""
		}
	}

	return scanner.Err()
}

// Bounded to GET, since we don't use anything for it right now (in the furute newRequest + newStream should be refactored)
func (c *Client) newStreamRequest(ctx context.Context, path string) (*http.Request, error) {
	url := baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request for %s: %w", path, err)
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	return req, nil
}

func (c *Client) newRequest(ctx context.Context, method, path string, in any) (*http.Request, error) {
	var body io.Reader
	if in != nil {
		buf, err := json.Marshal(in)
		if err != nil {
			return nil, fmt.Errorf("encode request body for %s: %w", path, err)
		}
		body = bytes.NewReader(buf)
	}

	url := baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("build request for %s: %w", path, err)
	}

	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := c.newRequest(ctx, method, path, in)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return DeamonError(err)
	}

	if err := checkStatus(resp); err != nil {
		return err
	}

	if out == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s response: %w", path, err)
	}

	return nil
}

func checkStatus(resp *http.Response) error {
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	raw, _ := io.ReadAll(resp.Body)
	msg := strings.TrimSpace(string(raw))

	return &ClientError{Code: resp.StatusCode, Body: msg}
}
