package porkbun

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/dnswlt/porkbun/pkg/api"
)

const (
	PorkbunApiV3Url     = "https://api.porkbun.com/api/json/v3/"
	PorkbunApiV3Ipv4Url = "https://api-ipv4.porkbun.com/api/json/v3/"
)

type Client struct {
	BaseURL string
	Config  *ClientConfig
	client  *http.Client
}

type ClientConfig struct {
	Domain string `json:"domain"`
	api.Keys
}

func ReadClientConfig(path string) (*ClientConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %v", err)
	}
	defer f.Close()
	config := &ClientConfig{}
	err = json.NewDecoder(f).Decode(config)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON: %v", err)
	}
	return config, nil
}

func NewClient(config *ClientConfig, useIPV4 bool) *Client {
	url := PorkbunApiV3Url
	if useIPV4 {
		url = PorkbunApiV3Ipv4Url
	}
	return &Client{
		BaseURL: url,
		Config:  config,
		client:  &http.Client{},
	}
}

func (c *Client) url(elem ...string) string {
	p, err := url.JoinPath(c.BaseURL, elem...)
	if err != nil {
		panic(fmt.Sprintf("Cannot join URL paths: %v %v", c.BaseURL, elem))
	}
	return p
}

func doRequest[Resp any, Req any](c *Client, ctx context.Context, url string, req *Req) (*Resp, error) {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(req)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal request: %v", err)
	}
	r, err := http.NewRequestWithContext(ctx, "POST", url, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	response, err := c.client.Do(r)
	if err != nil {
		return nil, fmt.Errorf("POST failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, fmt.Errorf("response status %s (could not read response body: %v)", response.Status, err)
		}
		return nil, fmt.Errorf("response status %s. Body: %v)", response.Status, string(body))
	}
	resp := new(Resp)
	err = json.NewDecoder(response.Body).Decode(resp)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal response: %v", err)
	}
	return resp, nil
}

func (c *Client) Ping(ctx context.Context) (*api.PingResponse, error) {
	req := api.PingRequest{
		Keys: c.Config.Keys,
	}
	return doRequest[api.PingResponse](c, ctx, c.url("ping"), &req)
}

func (c *Client) CreateA(ctx context.Context, subdomain string, ipv4Address string) (*api.CreateResponse, error) {
	req := api.UpdateRequest{
		Keys:    c.Config.Keys,
		Name:    subdomain,
		Type:    "A",
		Content: ipv4Address,
		// Use defaults for TTL and Prio
	}
	return doRequest[api.CreateResponse](c, ctx, c.url("dns/create"), &req)
}

func (c *Client) EditAllA(ctx context.Context, subdomain string, ipv4Address string) (*api.EditResponse, error) {
	req := api.UpdateRequest{
		Keys:    c.Config.Keys,
		Content: ipv4Address,
	}
	var u string
	if subdomain == "" {
		u = c.url("dns/editByNameType", c.Config.Domain, "A")
	} else {
		u = c.url("dns/editByNameType", c.Config.Domain, "A", subdomain)
	}
	return doRequest[api.EditResponse](c, ctx, u, &req)
}

func (c *Client) RetrieveAll(ctx context.Context) (*api.RecordsResponse, error) {
	req := api.RecordsRequest{
		Keys: c.Config.Keys,
	}
	url := c.url("dns/retrieve", c.Config.Domain)
	return doRequest[api.RecordsResponse](c, ctx, url, &req)
}
