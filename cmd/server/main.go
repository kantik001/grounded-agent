package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kantik001/grounded-agent/internal/api"
	"github.com/kantik001/grounded-agent/internal/config"
	"github.com/kantik001/grounded-agent/internal/guardrails"
	"github.com/kantik001/grounded-agent/internal/llm"
	"github.com/kantik001/grounded-agent/internal/mcp"
	"github.com/kantik001/grounded-agent/internal/memory"
	"github.com/kantik001/grounded-agent/internal/react"
	"github.com/kantik001/grounded-agent/internal/retrieve"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parseLevel(cfg.LogLevel)}))

	var mem react.Memory = memory.Nop{}
	if store, err := memory.New(cfg.RedisURL, cfg.MemoryMaxPairs, cfg.MemoryTTL); err != nil {
		log.Warn("redis unavailable; using nop memory", "err", err)
	} else {
		mem = store
		defer func() { _ = store.Close() }()
		log.Info("redis memory connected")
	}

	retriever := retrieve.Instrumented{Inner: retrieve.New(cfg.RetrieveMode, cfg.GroundedGRPCAddr, cfg.RAGHTTPURL, cfg.RAGServiceToken)}
	tools := mcp.Instrumented{Inner: mcp.NewClient(cfg.MCPGatewayURL, cfg.MCPAPIKey)}
	var completer llm.Completer = llm.NewOpenAIClient(cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.LLMModel)
	if cfg.LLMMode == "demo" {
		completer = &llm.DemoCompleter{}
		log.Info("LLM_MODE=demo (rule-based ReAct, no API key)")
	}

	var verifier react.AnswerVerifier
	mode := guardrails.NormalizeMode(cfg.GuardrailsMode)
	if mode != guardrails.ModeOff {
		adapter, err := guardrails.NewAdapter(mode, cfg.GuardrailsGRPCAddr)
		if err != nil {
			log.Error("guardrails", "err", err)
			os.Exit(1)
		}
		verifier = adapter
		log.Info("guardrails verify enabled", "mode", mode, "addr", cfg.GuardrailsGRPCAddr)
	}

	eng := &react.Engine{
		LLM:       completer,
		Retriever: retriever,
		Tools:     tools,
		Memory:    mem,
		Verifier:  verifier,
		MaxSteps:  cfg.ReactMaxSteps,
		DomainID:  cfg.DefaultDomainID,
		TenantID:  cfg.DefaultTenantID,
		Locale:    cfg.DefaultLocale,
	}

	srv := &api.Server{Engine: eng, Log: log}
	httpSrv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           srv.Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("listening", "addr", httpSrv.Addr, "retrieve_mode", cfg.RetrieveMode, "guardrails_mode", cfg.GuardrailsMode)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server", "err", err)
			os.Exit(1)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
	log.Info("shutdown complete")
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
