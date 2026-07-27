package react

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/kantik001/grounded-agent/internal/llm"
)

const needMoreInfo = "I need more information"

// Action kinds supported by the ReAct loop.
const (
	ActionRetrieve = "retrieve"
	ActionCallTool = "call_tool"
	ActionAnswer   = "answer"
)

// ParsedAction is one ReAct action extracted from an LLM turn.
type ParsedAction struct {
	Kind   string
	Query  string          // retrieve[...]
	Server string          // call_tool
	Tool   string          // call_tool
	Args   json.RawMessage // call_tool JSON args
	Answer string          // answer[...]
	Raw    string
}

// Step is one Thought/Action/Observation cycle.
type Step struct {
	Thought     string `json:"thought,omitempty"`
	ActionRaw   string `json:"action_raw,omitempty"`
	Action      string `json:"action,omitempty"`
	Observation string `json:"observation,omitempty"`
}

// Result is the final agent response.
type Result struct {
	Answer string `json:"answer"`
	Steps  []Step `json:"steps"`
	Limited bool  `json:"limited,omitempty"`
}

// Retriever fetches grounded context.
type Retriever interface {
	Retrieve(ctx context.Context, query, domainID, tenantID, locale string) (string, error)
}

// ToolCaller invokes MCP tools via the gateway.
type ToolCaller interface {
	CallTool(ctx context.Context, server, tool string, args json.RawMessage) (string, error)
	ToolCatalog(ctx context.Context) (string, error)
}

// Memory stores conversational turns.
type Memory interface {
	Load(ctx context.Context, sessionID string) (string, error)
	Append(ctx context.Context, sessionID, user, assistant string) error
}

// Engine runs the ReAct loop.
type Engine struct {
	LLM       llm.Completer
	Retriever Retriever
	Tools     ToolCaller
	Memory    Memory
	MaxSteps  int
	DomainID  string
	TenantID  string
	Locale    string
}

var (
	reThought = regexp.MustCompile(`(?is)Thought:\s*(.+?)(?:\n\s*Action:|$)`)
	reAction  = regexp.MustCompile(`(?is)Action:\s*(.+?)(?:\n\s*Observation:|$)`)
	reRetrieve = regexp.MustCompile(`(?is)^retrieve\[(.*)\]\s*$`)
	reAnswer   = regexp.MustCompile(`(?is)^answer\[(.*)\]\s*$`)
	reCallTool = regexp.MustCompile(`(?is)^call_tool\[\s*([a-zA-Z0-9_.-]+)\.([a-zA-Z0-9_.-]+)\s*(?:,\s*(.+))?\s*\]\s*$`)
)

// ParseAction parses a single Action: line body.
func ParseAction(raw string) (ParsedAction, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, "`")
	raw = strings.TrimSpace(raw)

	pa := ParsedAction{Raw: raw}
	if m := reRetrieve.FindStringSubmatch(raw); m != nil {
		pa.Kind = ActionRetrieve
		pa.Query = strings.TrimSpace(m[1])
		return pa, nil
	}
	if m := reAnswer.FindStringSubmatch(raw); m != nil {
		pa.Kind = ActionAnswer
		pa.Answer = strings.TrimSpace(m[1])
		return pa, nil
	}
	if m := reCallTool.FindStringSubmatch(raw); m != nil {
		pa.Kind = ActionCallTool
		pa.Server = m[1]
		pa.Tool = m[2]
		args := strings.TrimSpace(m[3])
		if args == "" {
			pa.Args = json.RawMessage(`{}`)
		} else {
			if !json.Valid([]byte(args)) {
				return pa, fmt.Errorf("call_tool args must be JSON object, got %q", args)
			}
			pa.Args = json.RawMessage(args)
		}
		return pa, nil
	}
	return pa, fmt.Errorf("unknown action %q (want retrieve[...], call_tool[server.tool, {...}], or answer[...])", raw)
}

// ParseTurn extracts Thought and Action from an LLM completion.
func ParseTurn(text string) (thought, action string) {
	if m := reThought.FindStringSubmatch(text); m != nil {
		thought = strings.TrimSpace(m[1])
	}
	if m := reAction.FindStringSubmatch(text); m != nil {
		action = strings.TrimSpace(m[1])
		// first line only for Action body when multiline
		if i := strings.IndexByte(action, '\n'); i >= 0 {
			action = strings.TrimSpace(action[:i])
		}
	}
	return thought, action
}

func systemPrompt(toolCatalog string) string {
	var b strings.Builder
	b.WriteString(`You are Grounded Agent, a ReAct orchestrator for document-grounded assistants.

You MUST reply using this exact format every turn:
Thought: <brief reasoning>
Action: <one action>

Allowed actions (exactly one per turn):
- retrieve[<search query>] — fetch grounded document chunks from Grounded LLM
- call_tool[<server>.<tool>, <json_args>] — call a tool via MCP Gateway (json_args optional, default {})
- answer[<final answer text>] — finish and return the answer to the user

Rules:
- Prefer retrieve before answering factual questions about policies/docs.
- Use call_tool only when needed for files or external tools.
- Never invent document facts; if retrieval is empty, say you do not know in answer[...].
- Keep answers concise and cite sources mentioned in observations when present.
`)
	if strings.TrimSpace(toolCatalog) != "" {
		b.WriteString("\nAvailable MCP tools:\n")
		b.WriteString(toolCatalog)
		b.WriteString("\n")
	}
	return b.String()
}

// Run executes the ReAct loop for one user query.
func (e *Engine) Run(ctx context.Context, sessionID, query string) (Result, error) {
	if e.MaxSteps < 1 {
		e.MaxSteps = 5
	}
	domain := e.DomainID
	if domain == "" {
		domain = "default"
	}
	tenant := e.TenantID
	if tenant == "" {
		tenant = "default"
	}
	locale := e.Locale
	if locale == "" {
		locale = "en"
	}

	catalog := ""
	if e.Tools != nil {
		if c, err := e.Tools.ToolCatalog(ctx); err == nil {
			catalog = c
		}
	}
	sys := systemPrompt(catalog)

	mem := ""
	if e.Memory != nil && sessionID != "" {
		if m, err := e.Memory.Load(ctx, sessionID); err == nil {
			mem = m
		}
	}

	var transcript strings.Builder
	transcript.WriteString("User question: ")
	transcript.WriteString(query)
	transcript.WriteString("\n")
	if mem != "" {
		transcript.WriteString("\nConversation memory:\n")
		transcript.WriteString(mem)
		transcript.WriteString("\n")
	}

	var steps []Step
	for i := 0; i < e.MaxSteps; i++ {
		userPrompt := transcript.String() + "\nProduce the next Thought and Action."
		completion, err := e.LLM.Complete(ctx, sys, userPrompt)
		if err != nil {
			return Result{}, err
		}
		thought, actionRaw := ParseTurn(completion)
		if actionRaw == "" {
			// try whole completion as action
			actionRaw = strings.TrimSpace(completion)
		}
		pa, err := ParseAction(actionRaw)
		step := Step{Thought: thought, ActionRaw: actionRaw, Action: pa.Kind}
		if err != nil {
			obs := "Error: " + err.Error()
			step.Observation = obs
			steps = append(steps, step)
			fmt.Fprintf(&transcript, "\nThought: %s\nAction: %s\nObservation: %s\n", thought, actionRaw, obs)
			continue
		}

		var obs string
		switch pa.Kind {
		case ActionRetrieve:
			if e.Retriever == nil {
				obs = "Error: retriever not configured"
			} else {
				ctxText, rerr := e.Retriever.Retrieve(ctx, pa.Query, domain, tenant, locale)
				if rerr != nil {
					obs = "Error: " + rerr.Error()
				} else if strings.TrimSpace(ctxText) == "" {
					obs = "No relevant documents found."
				} else {
					obs = ctxText
				}
			}
		case ActionCallTool:
			if e.Tools == nil {
				obs = "Error: tools not configured"
			} else {
				out, terr := e.Tools.CallTool(ctx, pa.Server, pa.Tool, pa.Args)
				if terr != nil {
					obs = "Error: " + terr.Error()
				} else {
					obs = out
				}
			}
		case ActionAnswer:
			step.Observation = "(final)"
			steps = append(steps, step)
			res := Result{Answer: pa.Answer, Steps: steps}
			if e.Memory != nil && sessionID != "" {
				_ = e.Memory.Append(ctx, sessionID, query, pa.Answer)
			}
			return res, nil
		default:
			obs = "Error: unsupported action"
		}
		step.Observation = obs
		steps = append(steps, step)
		fmt.Fprintf(&transcript, "\nThought: %s\nAction: %s\nObservation: %s\n", thought, actionRaw, truncateObs(obs))
	}

	res := Result{Answer: needMoreInfo, Steps: steps, Limited: true}
	if e.Memory != nil && sessionID != "" {
		_ = e.Memory.Append(ctx, sessionID, query, res.Answer)
	}
	return res, nil
}

func truncateObs(s string) string {
	const max = 4000
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n...[truncated]"
}
