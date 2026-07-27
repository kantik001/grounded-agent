package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kantik001/grounded-agent/internal/metrics"
	"github.com/kantik001/grounded-agent/internal/react"
)

// Server exposes HTTP endpoints for the agent.
type Server struct {
	Engine *react.Engine
	Log    *slog.Logger
}

type chatRequest struct {
	Query     string `json:"query"`
	SessionID string `json:"session_id"`
	DomainID  string `json:"domain_id,omitempty"`
	TenantID  string `json:"tenant_id,omitempty"`
	Locale    string `json:"locale,omitempty"`
}

type chatResponse struct {
	Answer  string       `json:"answer"`
	Steps   []react.Step `json:"steps"`
	Limited bool         `json:"limited,omitempty"`
}

// Router builds the chi router.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(3 * time.Minute))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"grounded-agent"}`))
	})
	r.Handle("/metrics", promhttp.Handler())
	r.Post("/agent/chat", s.handleChat)
	return r
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	metrics.ChatRequests.Inc()
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		metrics.ChatErrors.Inc()
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}
	if req.Query == "" {
		metrics.ChatErrors.Inc()
		http.Error(w, `{"error":"query is required"}`, http.StatusBadRequest)
		return
	}

	eng := *s.Engine
	if req.DomainID != "" {
		eng.DomainID = req.DomainID
	}
	if req.TenantID != "" {
		eng.TenantID = req.TenantID
	}
	if req.Locale != "" {
		eng.Locale = req.Locale
	}

	start := time.Now()
	res, err := eng.Run(r.Context(), req.SessionID, req.Query)
	_ = time.Since(start)
	if err != nil {
		metrics.ChatErrors.Inc()
		s.Log.Error("chat failed", "err", err)
		http.Error(w, `{"error":"`+escapeJSON(err.Error())+`"}`, http.StatusBadGateway)
		return
	}
	metrics.ReactSteps.Observe(float64(len(res.Steps)))
	for _, st := range res.Steps {
		if st.Action == react.ActionCallTool {
			// best-effort labels from raw action not parsed here
			metrics.ToolCalls.WithLabelValues("mcp", "call").Inc()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(chatResponse{
		Answer:  res.Answer,
		Steps:   res.Steps,
		Limited: res.Limited,
	})
}

func escapeJSON(s string) string {
	b, _ := json.Marshal(s)
	if len(b) >= 2 {
		return string(b[1 : len(b)-1])
	}
	return s
}
