package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/bak1an/artf/version"
)

const (
	controlSocketTimeout = 5 * time.Second
)

type AdminClient struct {
	socketPath string
	client     *http.Client
}

func NewAdminClient(dataDir string) (*AdminClient, error) {
	socketPath := filepath.Join(dataDir, controlSocket)
	client := &http.Client{
		Timeout: controlSocketTimeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				dialer := &net.Dialer{}
				return dialer.DialContext(ctx, "unix", socketPath)
			},
		},
	}
	return &AdminClient{socketPath: socketPath, client: client}, nil
}

func (c *AdminClient) Path() string {
	return c.socketPath
}

func (c *AdminClient) ListKeys() ([]*Key, error) {
	body, err := c.request("GET", "/keys", nil)
	if err != nil {
		return nil, fmt.Errorf("cannot list keys: %w", err)
	}
	var resp KeyListResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal keys list: %w", err)
	}
	return resp.Keys, nil
}

func (c *AdminClient) CreateKey(name string, readOnly bool) (*Key, error) {
	createReq := KeyCreateRequest{
		Name:     name,
		ReadOnly: readOnly,
	}
	requestBody, err := json.Marshal(createReq)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal create request: %w", err)
	}
	body, err := c.request("PUT", "/keys", requestBody)
	if err != nil {
		return nil, fmt.Errorf("cannot create key: %w", err)
	}
	var resp Key
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal create response: %w", err)
	}
	return &resp, nil
}

func (c *AdminClient) DeleteKey(id uint64) error {
	_, err := c.request("DELETE", "/keys/"+strconv.FormatUint(id, 10), nil)
	if err != nil {
		return fmt.Errorf("cannot delete key: %w", err)
	}
	return nil
}

func (c *AdminClient) Ping() error {
	body, err := c.request("GET", "/ping", nil)
	if err != nil {
		return fmt.Errorf("cannot ping admin server: %w", err)
	}
	if string(body) != "pong" {
		return fmt.Errorf("unexpected ping response: %s", string(body))
	}
	return nil
}

func (c *AdminClient) Version() (version.BuildInfo, error) {
	body, err := c.request("GET", "/version", nil)
	if err != nil {
		return version.BuildInfo{}, fmt.Errorf("cannot get running version: %w", err)
	}
	var v version.BuildInfo
	err = json.Unmarshal(body, &v)
	if err != nil {
		return version.BuildInfo{}, fmt.Errorf("cannot unmarshal version info: %w", err)
	}
	return v, nil
}

func (c *AdminClient) request(method string, path string, body []byte) ([]byte, error) {
	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequest(method, "http://unix"+path, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("cannot create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, "http://unix"+path, nil)
		if err != nil {
			return nil, fmt.Errorf("cannot create request: %w", err)
		}
	}

	resp, err := c.client.Do(req)
	if err != nil || resp == nil {
		return nil, fmt.Errorf("cannot %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cannot %s %s: %s", method, path, resp.Status)
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read response body: %w", err)
	}
	return body, nil
}
