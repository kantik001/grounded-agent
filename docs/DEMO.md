# Agent demo — HR retrieve + MCP ticket + OOS refusal (3 minutes)

Offline path uses **no LLM API key**: `LLM_MODE=demo` + mock retrieve + mock MCP gateway.

Document facts still come from Grounded-style retrieval (mock handbook here).  
Side effects go through MCP. **This agent does not replace Grounded Spec verify** — production verify stays in [grounded-llm](https://github.com/kantik001/grounded-llm).

---

## One-shot (`make demo`)

From `grounded-agent` root (Redis required for memory; demo works with nop if Redis down):

```bash
# terminal A
python scripts/mock_mcp_gateway.py

# terminal B
cp -n .env.example .env   # or copy on Windows
# ensure:
#   LLM_MODE=demo
#   RETRIEVE_MODE=mock
#   MCP_GATEWAY_URL=http://127.0.0.1:8080
docker compose up -d redis
go run ./cmd/server
```

Or: `make demo` (starts mock MCP in background when possible, then agent with demo env).

---

## Three curls

**1. KB fact (retrieve → answer)**

```bash
curl -s -X POST http://localhost:8000/agent/chat \
  -H "Content-Type: application/json" \
  -d '{"query":"How many paid vacation days?","session_id":"demo-kb"}'
```

Expect steps: `retrieve` then `answer` mentioning **28**.

**2. Side effect (MCP tool)**

```bash
curl -s -X POST http://localhost:8000/agent/chat \
  -H "Content-Type: application/json" \
  -d '{"query":"Create an HR ticket for leave approval","session_id":"demo-ticket"}'
```

Expect: `call_tool[hr.create_ticket, ...]` then answer with ticket id from mock gateway.

**3. Out of scope (refuse)**

```bash
curl -s -X POST http://localhost:8000/agent/chat \
  -H "Content-Type: application/json" \
  -d '{"query":"What is the CEO salary on the Moon in 2099?","session_id":"demo-oos"}'
```

Expect: refusal — no invented salary.

---

## Full stack (optional)

Live Grounded LLM images + real MCP:

```bash
# set LLM_API_KEY; sibling ../mcp-gateway
make docker-up-full
```

Images should track grounded-llm **0.4.0+**. Prefer the offline demo above for sales/interview reliability.
