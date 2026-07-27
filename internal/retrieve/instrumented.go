package retrieve

import (
	"context"
	"time"

	"github.com/kantik001/grounded-agent/internal/metrics"
)

// Instrumented wraps a Retriever with Prometheus latency.
type Instrumented struct {
	Inner interface {
		Retrieve(ctx context.Context, query, domainID, tenantID, locale string) (string, error)
	}
}

func (i Instrumented) Retrieve(ctx context.Context, query, domainID, tenantID, locale string) (string, error) {
	start := time.Now()
	out, err := i.Inner.Retrieve(ctx, query, domainID, tenantID, locale)
	metrics.RetrieveLatency.Observe(time.Since(start).Seconds())
	return out, err
}
