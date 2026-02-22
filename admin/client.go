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
		return nil, err
	}
	var resp KeyListResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	body, err := c.request("PUT", "/keys", requestBody)
	if err != nil {
		return nil, err
	}
	var resp Key
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *AdminClient) DeleteKey(id uint64) error {
	_, err := c.request("DELETE", "/keys/"+strconv.FormatUint(id, 10), nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *AdminClient) ListRepos() ([]*Repo, error) {
	body, err := c.request("GET", "/repos", nil)
	if err != nil {
		return nil, err
	}
	var resp RepoListResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Repos, nil
}

func (c *AdminClient) CreateRepo(name string, keepCount, keepDays int) (*Repo, error) {
	createReq := RepoCreateRequest{
		Name:      name,
		KeepCount: keepCount,
		KeepDays:  keepDays,
	}
	requestBody, err := json.Marshal(createReq)
	if err != nil {
		return nil, err
	}
	body, err := c.request("PUT", "/repos", requestBody)
	if err != nil {
		return nil, err
	}
	var resp Repo
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *AdminClient) DeleteRepo(id uint64) error {
	_, err := c.request("DELETE", "/repos/"+strconv.FormatUint(id, 10), nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *AdminClient) Ping() error {
	body, err := c.request("GET", "/ping", nil)
	if err != nil {
		return err
	}
	bodyStr := string(body)
	if bodyStr != "pong" {
		return fmt.Errorf("unexpected ping response: %s", bodyStr)
	}
	return nil
}

func (c *AdminClient) Version() (version.BuildInfo, error) {
	body, err := c.request("GET", "/version", nil)
	if err != nil {
		return version.BuildInfo{}, err
	}
	var v version.BuildInfo
	err = json.Unmarshal(body, &v)
	if err != nil {
		return version.BuildInfo{}, err
	}
	return v, nil
}

func (c *AdminClient) request(method string, path string, body []byte) ([]byte, error) {
	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequest(method, "http://unix"+path, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, "http://unix"+path, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := c.client.Do(req)
	if err != nil || resp == nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s %s returned %s", method, path, resp.Status)
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
