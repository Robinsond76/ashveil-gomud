package company

import (
	"math/rand"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/events"
)

// freshEvents drops events an earlier test queued and never processed (they
// would otherwise reach this test's listeners) and again when the test ends.
func freshEvents(t testing.TB) {
	t.Helper()
	events.ClearQueueForTest()
	t.Cleanup(events.ClearQueueForTest)
}

// seedDice makes the process-wide dice a fixed stream for one test (a
// golden record that needs the same rolls every run) and, when the test
// ends, hands them back to a fresh seed. rand.Seed replaces the global
// source for good, so a test that seeds without undoing it leaves every
// later test the rest of that stream, from wherever this one stopped: what
// a fight rolls then depends on which tests ran before it, and a failure
// seen under one shuffle cannot be repeated under another (Phase 83).
// GODEBUG=randseednop=0 is needed for the seed to take (Go 1.24 makes
// rand.Seed a no-op otherwise); its cleanup is registered first so it is
// undone last, after the reseed.
func seedDice(t testing.TB, seed int64) {
	t.Helper()
	t.Setenv("GODEBUG", "randseednop=0")
	t.Cleanup(func() { rand.Seed(time.Now().UnixNano()) })
	rand.Seed(seed)
}
