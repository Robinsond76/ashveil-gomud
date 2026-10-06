package encumbrance

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
)

// freshEvents drops events an earlier test queued and never processed (they
// would otherwise reach this test's listeners) and again when the test ends.
func freshEvents(t testing.TB) {
	t.Helper()
	events.ClearQueueForTest()
	t.Cleanup(events.ClearQueueForTest)
}
