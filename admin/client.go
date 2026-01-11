package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
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

func (c *AdminClient) Ping() error {
	body, err := c.get("/ping")
	if err != nil {
		return fmt.Errorf("cannot ping admin server: %w", err)
	}
	if string(body) != "pong" {
		return fmt.Errorf("unexpected ping response: %s", string(body))
	}
	return nil
}

func (c *AdminClient) Version() (version.BuildInfo, error) {
	body, err := c.get("/version")
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

func (c *AdminClient) get(path string) ([]byte, error) {
	resp, err := c.client.Get("http://unix" + path)
	if err != nil {
		return nil, fmt.Errorf("cannot get %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cannot get %s: %s", path, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read response body: %w", err)
	}
	return body, nil
}
