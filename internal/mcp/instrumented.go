package mcp

import (
	"context"
	"encoding/json"

	"github.com/kantik001/grounded-agent/internal/metrics"
)

// Instrumented wraps Client with Prometheus counters.
type Instrumented struct {
	Inner *Client
}

func (i Instrumented) ToolCatalog(ctx context.Context) (string, error) {
	return i.Inner.ToolCatalog(ctx)
}

func (i Instrumented) CallTool(ctx context.Context, server, tool string, args json.RawMessage) (string, error) {
	out, err := i.Inner.CallTool(ctx, server, tool, args)
	metrics.ToolCalls.WithLabelValues(server, tool).Inc()
	return out, err
}
