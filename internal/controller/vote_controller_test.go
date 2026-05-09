package controller

import (
	"testing"

	"github.com/apache/answer/pkg/uid"
)

func TestDecodeVoteObjectID(t *testing.T) {
	rawID := "10020000000000144"
	if got := decodeVoteObjectID(rawID); got != rawID {
		t.Fatalf("raw ID should be preserved, got %q", got)
	}

	shortID := uid.EnShortID(rawID)
	if shortID == rawID {
		t.Fatalf("test setup failed: expected short ID, got %q", shortID)
	}
	if got := decodeVoteObjectID(shortID); got != rawID {
		t.Fatalf("short ID should decode to raw ID, got %q", got)
	}

	if got := decodeVoteObjectID(""); got != "" {
		t.Fatalf("empty ID should stay empty, got %q", got)
	}
}
