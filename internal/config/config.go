package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds runtime settings for grounded-agent.
type Config struct {
	Port     string
	LogLevel string

	LLMBaseURL string
	LLMAPIKey  string
	LLMModel   string
	LLMMode    string // openai | demo

	RetrieveMode      string // grpc | http | mock
	GroundedGRPCAddr  string
	RAGHTTPURL        string
	RAGServiceToken   string
	DefaultDomainID   string
	DefaultTenantID   string
	DefaultLocale     string

	MCPGatewayURL string
	MCPAPIKey     string

	RedisURL         string
	MemoryMaxPairs   int
	MemoryTTL        time.Duration

	ReactMaxSteps int

	GuardrailsMode     string // off | remote | hybrid
	GuardrailsGRPCAddr string
}

// Load reads .env (if present) and environment variables.
func Load() (Config, error) {
	_ = godotenv.Load()

	cfg := Config{
		Port:             env("PORT", "8000"),
		LogLevel:         env("LOG_LEVEL", "info"),
		LLMBaseURL:       strings.TrimRight(env("LLM_BASE_URL", "https://openrouter.ai/api"), "/"),
		LLMAPIKey:        env("LLM_API_KEY", ""),
		LLMModel:         env("LLM_MODEL", "openrouter/free"),
		LLMMode:          strings.ToLower(env("LLM_MODE", "openai")),
		RetrieveMode:     strings.ToLower(env("RETRIEVE_MODE", "grpc")),
		GroundedGRPCAddr: env("GROUNDED_GRPC_ADDR", "localhost:50051"),
		RAGHTTPURL:       strings.TrimRight(env("RAG_HTTP_URL", "http://localhost:5000"), "/"),
		RAGServiceToken:  env("RAG_SERVICE_TOKEN", ""),
		DefaultDomainID:  env("DEFAULT_DOMAIN_ID", "default"),
		DefaultTenantID:  env("DEFAULT_TENANT_ID", "default"),
		DefaultLocale:    env("DEFAULT_LOCALE", "en"),
		MCPGatewayURL:    strings.TrimRight(env("MCP_GATEWAY_URL", "http://localhost:8080"), "/"),
		MCPAPIKey:        env("MCP_API_KEY", ""),
		RedisURL:         env("REDIS_URL", "redis://localhost:6379/0"),
		MemoryMaxPairs:   envInt("MEMORY_MAX_PAIRS", 10),
		MemoryTTL:        time.Duration(envInt("MEMORY_TTL_MINUTES", 30)) * time.Minute,
		ReactMaxSteps:      envInt("REACT_MAX_STEPS", 5),
		GuardrailsMode:     strings.ToLower(env("GUARDRAILS_MODE", "off")),
		GuardrailsGRPCAddr: env("GUARDRAILS_GRPC_ADDR", "localhost:50052"),
	}

	switch cfg.LLMMode {
	case "openai", "demo":
	default:
		return cfg, fmt.Errorf("LLM_MODE must be openai|demo, got %q", cfg.LLMMode)
	}
	switch cfg.RetrieveMode {
	case "grpc", "http", "mock":
	default:
		return cfg, fmt.Errorf("RETRIEVE_MODE must be grpc|http|mock, got %q", cfg.RetrieveMode)
	}
	switch cfg.GuardrailsMode {
	case "", "off", "remote", "hybrid":
		if cfg.GuardrailsMode == "" {
			cfg.GuardrailsMode = "off"
		}
	default:
		return cfg, fmt.Errorf("GUARDRAILS_MODE must be off|remote|hybrid, got %q", cfg.GuardrailsMode)
	}
	if cfg.ReactMaxSteps < 1 {
		cfg.ReactMaxSteps = 5
	}
	return cfg, nil
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
