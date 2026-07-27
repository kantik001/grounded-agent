package retrieve

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pb "github.com/kantik001/grounded-agent/internal/retrieve/pb"
)

// Client retrieves grounded context via gRPC, HTTP, or mock.
type Client struct {
	Mode        string // grpc | http | mock
	GRPCAddr    string
	HTTPURL     string
	Token       string
	HTTPClient  *http.Client
	MockContext string
}

func New(mode, grpcAddr, httpURL, token string) *Client {
	return &Client{
		Mode:     mode,
		GRPCAddr: grpcAddr,
		HTTPURL:  strings.TrimRight(httpURL, "/"),
		Token:    token,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		MockContext: "Mock handbook: Employees receive 28 paid vacation days per year. Source: hr_policy.txt",
	}
}

// Retrieve returns a formatted observation string.
func (c *Client) Retrieve(ctx context.Context, query, domainID, tenantID, locale string) (string, error) {
	switch c.Mode {
	case "mock":
		return c.MockContext, nil
	case "http":
		return c.retrieveHTTP(ctx, query, domainID, tenantID, locale)
	default:
		return c.retrieveGRPC(ctx, query, domainID, tenantID, locale)
	}
}

func (c *Client) retrieveHTTP(ctx context.Context, query, domainID, tenantID, locale string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"query":     query,
		"domain_id": domainID,
		"tenant_id": tenantID,
		"locale":    locale,
		"top_k":     4,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.HTTPURL+"/rag/context", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("X-RAG-Service-Token", c.Token)
	}
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
		return "", fmt.Errorf("rag http %d: %s", resp.StatusCode, string(raw))
	}
	var out struct {
		Context string `json:"context"`
		Chunks  []struct {
			Text   string  `json:"text"`
			Source string  `json:"source"`
			Score  float64 `json:"score"`
		} `json:"chunks"`
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return string(raw), nil
	}
	if out.Error != "" {
		return "", fmt.Errorf("%s", out.Error)
	}
	if out.Context != "" {
		return out.Context, nil
	}
	var b strings.Builder
	for _, ch := range out.Chunks {
		if ch.Source != "" {
			fmt.Fprintf(&b, "Source: %s\n", ch.Source)
		}
		b.WriteString(ch.Text)
		b.WriteString("\n---\n")
	}
	return b.String(), nil
}

func (c *Client) retrieveGRPC(ctx context.Context, query, domainID, tenantID, locale string) (string, error) {
	conn, err := grpc.NewClient(c.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return "", err
	}
	defer conn.Close()

	cli := pb.NewRetrieverClient(conn)
	if c.Token != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-rag-service-token", c.Token)
	}
	resp, err := cli.Retrieve(ctx, &pb.RetrieveRequest{
		Query:    query,
		DomainId: domainID,
		TenantId: tenantID,
		Locale:   locale,
		TopK:     4,
	})
	if err != nil {
		return "", err
	}
	if !resp.Success && resp.Error != "" {
		return "", fmt.Errorf("%s", resp.Error)
	}
	if resp.Context != "" {
		return resp.Context, nil
	}
	var b strings.Builder
	for _, ch := range resp.Chunks {
		if ch.Source != "" {
			fmt.Fprintf(&b, "Source: %s\n", ch.Source)
		}
		b.WriteString(ch.Text)
		b.WriteString("\n---\n")
	}
	return b.String(), nil
}
