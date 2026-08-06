package llm

import (
	"context"
	"strings"
	"testing"
)

func TestDemoCompleterVacation(t *testing.T) {
	d := &DemoCompleter{}
	user := "User question: How many vacation days?\n\nProduce the next Thought and Action."
	out, err := d.Complete(context.Background(), "", user)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "retrieve[") {
		t.Fatalf("want retrieve, got %q", out)
	}
	user2 := user + "\nThought: x\nAction: retrieve[vacation]\nObservation: Mock handbook: 28 days\n\nProduce the next Thought and Action."
	out2, err := d.Complete(context.Background(), "", user2)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out2, "answer[") || !strings.Contains(out2, "28") {
		t.Fatalf("want answer with 28, got %q", out2)
	}
}

func TestDemoCompleterOOS(t *testing.T) {
	d := &DemoCompleter{}
	out, err := d.Complete(context.Background(), "", "User question: CEO salary on the Moon?\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "answer[") || !strings.Contains(out, "No information") {
		t.Fatalf("want refusal, got %q", out)
	}
}

func TestDemoCompleterTicket(t *testing.T) {
	d := &DemoCompleter{}
	out, err := d.Complete(context.Background(), "", "User question: Create an HR ticket for leave\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "call_tool[hr.create_ticket") {
		t.Fatalf("want call_tool, got %q", out)
	}
}
