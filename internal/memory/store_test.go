package memory

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestStoreRoundTrip(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	store, err := New("redis://"+mr.Addr()+"/0", 2, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.Append(ctx, "abc", "q1", "a1"); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(ctx, "abc", "q2", "a2"); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(ctx, "abc", "q3", "a3"); err != nil {
		t.Fatal(err)
	}
	mem, err := store.Load(ctx, "abc")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(mem, "q1") {
		t.Fatalf("expected trim of oldest pair, got %q", mem)
	}
	if !strings.Contains(mem, "q2") || !strings.Contains(mem, "q3") {
		t.Fatalf("memory=%q", mem)
	}
}
