package main

import (
	"testing"
	"time"
)

// TestWaitForTurnPacesInsteadOfDropping: found live (Phase 44), a line typed
// within one turn of the last was silently discarded. The connection loops
// now hold such a line until the turn has passed and then send it.
func TestWaitForTurnPacesInsteadOfDropping(t *testing.T) {
	turn := 40 * time.Millisecond

	start := time.Now()
	waitForTurn(start, turn)
	if got := time.Since(start); got < turn {
		t.Fatalf("a line right after the last waited %v, want at least %v", got, turn)
	}

	start = time.Now()
	waitForTurn(start.Add(-time.Second), turn)
	if got := time.Since(start); got >= turn {
		t.Fatalf("a line a second after the last waited %v, want no wait", got)
	}
}
