package market

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// haggleProvider is an archetypes provider whose only specialist is a
// haggler of a given level for user 7, in the market room.
type haggleProvider struct {
	level int
	asked *[]int
}

func (haggleProvider) CanTrain(int, string) (bool, string)      { return true, "" }
func (haggleProvider) CanLearnSpell(int, string) (bool, string) { return true, "" }
func (haggleProvider) Exists(string) bool                        { return false }
func (haggleProvider) ArchetypeName(string) (string, bool)       { return "", false }
func (haggleProvider) PlayerArchetype(int) (string, bool)        { return "", false }
func (p haggleProvider) BestSpecialist(leader int, utility string, roomIDs ...int) (archetypes.Specialist, bool) {
	if p.asked != nil {
		*p.asked = roomIDs
	}
	if leader != 7 || utility != archetypes.UtilityHaggle || p.level == 0 {
		return archetypes.Specialist{}, false
	}
	return archetypes.Specialist{Name: "Mira", Level: p.level}, true
}
func (haggleProvider) SpecialistsView(int) string { return "" }

func withHaggler(t *testing.T, level int, asked *[]int) {
	t.Helper()
	archetypes.SetProvider(haggleProvider{level: level, asked: asked})
	t.Cleanup(func() { archetypes.SetProvider(nil) })
}

// TestHagglerLowersTheBuyingPrice (33f2): a level-4 haggler takes 8% off,
// rounded up, and is named; the haggler must be at the leader's side.
func TestHagglerLowersTheBuyingPrice(t *testing.T) {
	w := newTradeWorld(t, 100)
	var asked []int
	withHaggler(t, 4, &asked)

	out := w.run(t, "buy hide")
	assert.Contains(t, out, "You buy the wolf hide at the market for 25 gold. Mira haggles 8% off the price.", "27 less 8% is 24.84, rounded up")
	assert.Equal(t, 75, w.user.Character.Gold)
	assert.Equal(t, []int{marketRoom().RoomId}, asked)
}

// TestHaggledRoundTripNeverProfits (33f2): buying and selling straight back
// with the best haggler still loses gold, at every haggler level.
func TestHaggledRoundTripNeverProfits(t *testing.T) {
	for level := 1; level <= 4; level++ {
		w := newTradeWorld(t, 100)
		withHaggler(t, level, nil)
		w.run(t, "buy hide")
		require.Equal(t, 1, countItem(w.user.Character.Items, 28))
		out := w.run(t, "sell hide")
		assert.Contains(t, out, "Mira haggles up to", "level %d", level)
		assert.Less(t, w.user.Character.Gold, 100, "level %d: a haggled round trip loses gold", level)
	}
}

// TestHaggledListingShowsTheBetterPrices (33f2): the price list shows the
// haggled prices the trade will charge, and says who haggles.
func TestHaggledListingShowsTheBetterPrices(t *testing.T) {
	w := newTradeWorld(t, 100)
	withHaggler(t, 4, nil)
	out := w.run(t, "")
	assert.Regexp(t, `wolf hide\s+25 gold`, out)
	assert.Contains(t, out, "Mira haggles for your company: these prices are up to 8% better")

	archetypes.SetProvider(haggleProvider{})
	out = w.run(t, "")
	assert.Regexp(t, `wolf hide\s+27 gold`, out)
	assert.NotContains(t, out, "haggles")
}
