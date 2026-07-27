package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client talks to mcp-gateway HTTP API.
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

type schemaResp struct {
	Tools []struct {
		Server  string `json:"server"`
		MCPTool string `json:"mcp_tool"`
		Type    string `json:"type"`
		Function struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"function"`
	} `json:"tools"`
}

// ToolCatalog returns a short text catalog for the ReAct system prompt.
func (c *Client) ToolCatalog(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/v1/tools/schema", nil)
	if err != nil {
		return "", err
	}
	c.auth(req)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("mcp schema http %d: %s", resp.StatusCode, string(raw))
	}
	var out schemaResp
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	var b strings.Builder
	for _, t := range out.Tools {
		server := t.Server
		tool := t.MCPTool
		if tool == "" && t.Function.Name != "" {
			tool = t.Function.Name
		}
		desc := t.Function.Description
		if server == "" || tool == "" {
			continue
		}
		fmt.Fprintf(&b, "- %s.%s", server, tool)
		if desc != "" {
			fmt.Fprintf(&b, ": %s", desc)
		}
		b.WriteByte('\n')
	}
	return b.String(), nil
}

type callBody struct {
	Args json.RawMessage `json:"args"`
}

// CallTool invokes POST /v1/servers/{server}/tools/{tool}.
func (c *Client) CallTool(ctx context.Context, server, tool string, args json.RawMessage) (string, error) {
	if len(args) == 0 {
		args = json.RawMessage(`{}`)
	}
	body, err := json.Marshal(callBody{Args: args})
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("%s/v1/servers/%s/tools/%s", c.BaseURL, server, tool)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	c.auth(req)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("mcp call http %d: %s", resp.StatusCode, string(raw))
	}
	return string(raw), nil
}

func (c *Client) auth(req *http.Request) {
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("X-API-Key", c.APIKey)
	}
}
