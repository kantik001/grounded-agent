# Changelog

All notable changes to this project are documented in this file.

## [Unreleased]

### Added

- Offline demo path: `LLM_MODE=demo`, `scripts/mock_mcp_gateway.py`, [docs/DEMO.md](docs/DEMO.md), `make demo-curls`
- Compose full profile images bumped to grounded-llm **0.4.0**

## [0.1.0] - 2026-07-27

Initial public release of the Grounded Agent ReAct orchestrator.

### Added

- Go ReAct engine (`Thought` / `Action` / `Observation`) with max 5 steps
- Actions: `retrieve[...]`, `call_tool[server.tool, {...}]`, `answer[...]`
- Grounded LLM retrieval client: gRPC (`grounded.rag.v1.Retriever`), HTTP `/rag/context`, mock mode
- MCP Gateway client: `GET /v1/tools/schema`, `POST /v1/servers/{server}/tools/{tool}`
- Redis session memory (`session:{id}`, 10 pairs, TTL 30m)
- HTTP API: `POST /agent/chat`, `GET /health`, `GET /metrics`
- Docker Compose: default (agent + Redis), `--profile full` (Grounded LLM GHCR + MCP Gateway)
- CI: go test, vet, golangci-lint, docker build

[Unreleased]: https://github.com/kantik001/grounded-agent/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/kantik001/grounded-agent/releases/tag/v0.1.0
