package guardrails

import (
	"context"
	"fmt"
	"strings"
	"time"

	pb "github.com/kantik001/grounded-guardrails/go/gen/guardrails/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Mode controls optional final-answer verification.
type Mode string

const (
	ModeOff    Mode = "off"
	ModeRemote Mode = "remote"
	ModeHybrid Mode = "hybrid"
)

// NormalizeMode maps env strings to Mode (default off).
func NormalizeMode(s string) Mode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "remote":
		return ModeRemote
	case "hybrid":
		return ModeHybrid
	default:
		return ModeOff
	}
}

// Verdict is the result of VerifyText.
type Verdict struct {
	Passed     bool
	Violations []string
	LatencyMS  float32
}

// Client calls grounded-guardrails GuardrailsService.
type Client struct {
	addr    string
	timeout time.Duration
}

// NewClient returns a dial-per-call client (same pattern as retrieve).
func NewClient(addr string) *Client {
	if strings.TrimSpace(addr) == "" {
		addr = "localhost:50052"
	}
	return &Client{addr: addr, timeout: 3 * time.Second}
}

// VerifyText runs unary verify against context (retrieval observations).
func (c *Client) VerifyText(ctx context.Context, text, retrievalContext, tenantID string) (Verdict, error) {
	dialCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	conn, err := grpc.NewClient(c.addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return Verdict{}, fmt.Errorf("dial guardrails %s: %w", c.addr, err)
	}
	defer conn.Close()

	client := pb.NewGuardrailsServiceClient(conn)
	resp, err := client.VerifyText(dialCtx, &pb.TextRequest{
		Text:     text,
		Context:  retrievalContext,
		TenantId: tenantID,
	})
	if err != nil {
		return Verdict{}, fmt.Errorf("VerifyText: %w", err)
	}
	return Verdict{
		Passed:     resp.GetPassed(),
		Violations: append([]string(nil), resp.GetViolations()...),
		LatencyMS:  resp.GetLatencyMs(),
	}, nil
}
