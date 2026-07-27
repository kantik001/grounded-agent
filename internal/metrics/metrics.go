package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ChatRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "agent_chat_requests_total",
		Help: "Total /agent/chat requests",
	})
	ChatErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "agent_chat_errors_total",
		Help: "Total /agent/chat errors",
	})
	ReactSteps = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "agent_react_steps",
		Help:    "ReAct steps per chat request",
		Buckets: []float64{1, 2, 3, 4, 5, 6, 8, 10},
	})
	ToolCalls = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "agent_tool_calls_total",
		Help: "MCP tool calls from the agent",
	}, []string{"server", "tool"})
	RetrieveLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "agent_retrieve_seconds",
		Help:    "Grounded retrieve latency",
		Buckets: prometheus.DefBuckets,
	})
)
