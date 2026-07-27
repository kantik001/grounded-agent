package guardrails

import (
	"context"
	"testing"
)

func TestAdapterOff(t *testing.T) {
	a := Adapter{Mode: ModeOff, Client: NewClient("localhost:50052")}
	ok, _, err := a.VerifyText(context.Background(), "x", "ctx", "t")
	if err != nil || !ok {
		t.Fatalf("off should pass: ok=%v err=%v", ok, err)
	}
}

func TestAdapterNilClient(t *testing.T) {
	a := Adapter{Mode: ModeRemote, Client: nil}
	ok, _, err := a.VerifyText(context.Background(), "x", "", "")
	if err != nil || !ok {
		t.Fatalf("nil client should pass")
	}
}

func TestNormalizeMode(t *testing.T) {
	if NormalizeMode("REMOTE") != ModeRemote {
		t.Fatal()
	}
	if NormalizeMode("") != ModeOff {
		t.Fatal()
	}
}

func TestNewAdapterOff(t *testing.T) {
	a, err := NewAdapter(ModeOff, "")
	if err != nil || a != nil {
		t.Fatalf("want nil adapter: %v %v", a, err)
	}
}

func TestHybridTransportError(t *testing.T) {
	a := &Adapter{Mode: ModeHybrid, Client: NewClient("127.0.0.1:1")}
	ok, _, err := a.VerifyText(context.Background(), "text", "ctx", "t")
	if err != nil {
		t.Fatalf("hybrid should soft-skip: %v", err)
	}
	if !ok {
		t.Fatal("hybrid soft-skip should pass")
	}
}

func TestRemoteTransportError(t *testing.T) {
	a := &Adapter{Mode: ModeRemote, Client: NewClient("127.0.0.1:1")}
	ok, _, err := a.VerifyText(context.Background(), "text", "ctx", "t")
	if err == nil || ok {
		t.Fatalf("remote should hard-fail: ok=%v err=%v", ok, err)
	}
}
