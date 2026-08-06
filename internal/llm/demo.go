package llm

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

var (
	reUserQuestion = regexp.MustCompile(`(?m)^User question:\s*(.+)$`)
	reObservation  = regexp.MustCompile(`(?m)^Observation:`)
)

// DemoCompleter is a rule-based ReAct policy for offline demos (no LLM_API_KEY).
// Supports three canned scenarios: KB vacation fact, HR ticket via MCP, OOS refusal.
type DemoCompleter struct{}

func (d *DemoCompleter) Complete(ctx context.Context, system, user string) (string, error) {
	_ = ctx
	_ = system
	q := extractQuestion(user)
	steps := len(reObservation.FindAllStringIndex(user, -1))
	ql := strings.ToLower(q)

	switch {
	case isOOS(ql):
		return "Thought: Out of knowledge base; refuse without inventing facts.\nAction: answer[No information found in the knowledge base for this question.]", nil

	case isTicket(ql):
		if steps == 0 {
			title := "HR request"
			if i := strings.Index(ql, "ticket"); i >= 0 {
				title = strings.TrimSpace(q)
			}
			return fmt.Sprintf(
				"Thought: Need a side effect via MCP, not document retrieval.\nAction: call_tool[hr.create_ticket, {\"title\":%q,\"priority\":\"normal\"}]",
				title,
			), nil
		}
		return "Thought: Ticket created; summarize for the user.\nAction: answer[Created HR ticket (see tool observation). Document facts still come from retrieve when needed.]", nil

	default:
		// Default HR vacation / policy demo
		if steps == 0 {
			return "Thought: Need grounded policy facts from the knowledge base.\nAction: retrieve[paid vacation days]", nil
		}
		return "Thought: Use only retrieved handbook facts.\nAction: answer[Employees receive 28 paid vacation days per year (source: hr_policy.txt).]", nil
	}
}

func extractQuestion(user string) string {
	if m := reUserQuestion.FindStringSubmatch(user); m != nil {
		return strings.TrimSpace(m[1])
	}
	return strings.TrimSpace(user)
}

func isOOS(ql string) bool {
	for _, k := range []string{"moon", "mars", "neptune", "stock ticker", "2099"} {
		if strings.Contains(ql, k) {
			return true
		}
	}
	return false
}

func isTicket(ql string) bool {
	return strings.Contains(ql, "ticket") || strings.Contains(ql, "create hr") || strings.Contains(ql, "open a request")
}
