package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
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
