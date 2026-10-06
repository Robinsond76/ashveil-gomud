package camping

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/stretchr/testify/assert"
)

// scribeStub is a provider that reads gear (Phase 36a): the camping module
// asks it, through internal/archetypes, when a camp rest completes.
type scribeStub struct{ calls []int }

func (*scribeStub) CanTrain(int, string) (bool, string)      { return true, "" }
func (*scribeStub) CanLearnSpell(int, string) (bool, string) { return true, "" }
func (*scribeStub) Exists(string) bool                       { return false }
func (*scribeStub) ArchetypeName(string) (string, bool)      { return "", false }
func (*scribeStub) PlayerArchetype(int) (string, bool)       { return "", false }

func (s *scribeStub) CampIdentify(leaderUserID int) []string {
	s.calls = append(s.calls, leaderUserID)
	return []string{"You read the company's new finds: a stub blade."}
}

// TestCampRestCompletionAsksTheScribeOnce: the NewRound grant pass that
// closes a camp rest identifies the company's gear once, and tells the
// leader what was read.
func TestCampRestCompletionAsksTheScribeOnce(t *testing.T) {
	module, user, _, messages := autoSharpenCamp(t, false)
	stub := &scribeStub{}
	archetypes.SetProvider(stub)
	t.Cleanup(func() { archetypes.SetProvider(nil) })

	module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Equal(t, []int{user.UserId}, stub.calls)
	module.onNewRound(events.NewRound{RoundNumber: 2})
	assert.Equal(t, []int{user.UserId}, stub.calls, "once per rest")
	events.ProcessEvents()
	assert.Contains(t, strings.Join(*messages, "\n"), "You read the company's new finds")
}
