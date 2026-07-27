package react

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/kantik001/grounded-agent/internal/llm"
)

func TestParseAction(t *testing.T) {
	cases := []struct {
		in   string
		kind string
	}{
		{`retrieve[vacation days]`, ActionRetrieve},
		{`answer[Employees get 28 paid vacation days.]`, ActionAnswer},
		{`call_tool[filesystem.read_file, {"path":"/data/README.md"}]`, ActionCallTool},
		{`call_tool[filesystem.read_file]`, ActionCallTool},
	}
	for _, tc := range cases {
		pa, err := ParseAction(tc.in)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if pa.Kind != tc.kind {
			t.Fatalf("%q: kind=%s want %s", tc.in, pa.Kind, tc.kind)
		}
	}
}

type fakeRet struct{ text string }

func (f fakeRet) Retrieve(ctx context.Context, query, domainID, tenantID, locale string) (string, error) {
	return f.text, nil
}

type fakeTools struct{}

func (fakeTools) CallTool(ctx context.Context, server, tool string, args json.RawMessage) (string, error) {
	return `{"ok":true,"server":"` + server + `","tool":"` + tool + `"}`, nil
}
func (fakeTools) ToolCatalog(ctx context.Context) (string, error) {
	return "- filesystem.read_file", nil
}

func TestEngineRetrieveThenAnswer(t *testing.T) {
	eng := &Engine{
		LLM: &llm.ScriptedCompleter{Replies: []string{
			"Thought: need docs\nAction: retrieve[vacation days]",
			"Thought: have context\nAction: answer[28 paid vacation days per year.]",
		}},
		Retriever: fakeRet{text: "Source: handbook.txt\nEmployees receive 28 paid vacation days."},
		Tools:     fakeTools{},
		MaxSteps:  5,
	}
	res, err := eng.Run(context.Background(), "s1", "How many vacation days?")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Answer, "28") {
		t.Fatalf("answer=%q", res.Answer)
	}
	if len(res.Steps) != 2 {
		t.Fatalf("steps=%d", len(res.Steps))
	}
}

func TestEngineMaxSteps(t *testing.T) {
	eng := &Engine{
		LLM: &llm.ScriptedCompleter{Replies: []string{
			"Thought: a\nAction: retrieve[q]",
			"Thought: b\nAction: retrieve[q]",
			"Thought: c\nAction: retrieve[q]",
		}},
		Retriever: fakeRet{text: "x"},
		MaxSteps:  3,
	}
	res, err := eng.Run(context.Background(), "", "q")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Limited || res.Answer != needMoreInfo {
		t.Fatalf("got %+v", res)
	}
}

type fakeVerifier struct {
	passed     bool
	violations []string
	err        error
	sawCtx     string
	calls      int
}

func (f *fakeVerifier) VerifyText(ctx context.Context, text, retrievalContext, tenantID string) (bool, []string, error) {
	f.calls++
	f.sawCtx = retrievalContext
	return f.passed, f.violations, f.err
}

func TestEngineVerifyBlocksAnswer(t *testing.T) {
	v := &fakeVerifier{passed: false, violations: []string{"numeric claim not in context"}}
	eng := &Engine{
		LLM: &llm.ScriptedCompleter{Replies: []string{
			"Thought: need docs\nAction: retrieve[vacation]",
			"Thought: invent\nAction: answer[Employees get 99 vacation days.]",
		}},
		Retriever: fakeRet{text: "Source: handbook.txt\nEmployees receive 28 paid vacation days."},
		Verifier:  v,
		MaxSteps:  5,
	}
	res, err := eng.Run(context.Background(), "", "How many?")
	if err != nil {
		t.Fatal(err)
	}
	if v.calls != 1 {
		t.Fatalf("verify calls=%d", v.calls)
	}
	if !strings.Contains(v.sawCtx, "28 paid") {
		t.Fatalf("expected retrieval context, got %q", v.sawCtx)
	}
	if !strings.Contains(res.Answer, "Grounded verify failed") {
		t.Fatalf("answer=%q", res.Answer)
	}
	if !strings.Contains(res.Answer, "99 vacation") {
		t.Fatalf("blocked draft missing: %q", res.Answer)
	}
}

func TestEngineVerifyNilSkipped(t *testing.T) {
	eng := &Engine{
		LLM: &llm.ScriptedCompleter{Replies: []string{
			"Thought: done\nAction: answer[ok]",
		}},
		MaxSteps: 2,
	}
	res, err := eng.Run(context.Background(), "", "q")
	if err != nil {
		t.Fatal(err)
	}
	if res.Answer != "ok" {
		t.Fatalf("answer=%q", res.Answer)
	}
}
