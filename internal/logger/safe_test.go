package logger

import (
	"testing"
)

func TestRef_isStableAndOpaque(t *testing.T) {
	a := Ref("user", "550e8400-e29b-41d4-a716-446655440000")
	b := Ref("user", "550e8400-e29b-41d4-a716-446655440000")
	c := Ref("user", "other-id")

	if a.Value.String() != b.Value.String() {
		t.Fatal("expected stable hash for same id")
	}
	if a.Value.String() == c.Value.String() {
		t.Fatal("expected different hash for different id")
	}
	if got := a.Value.String(); len(got) != 8 {
		t.Fatalf("ref length = %d, want 8 hex chars", len(got))
	}
}
